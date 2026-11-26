# prov-ids-arreglo

Tipo: bug

## Objetivo

Corregir los dos fallos del manejo de identificadores que aparecieron al archivar C-03-c (adopcion-ids), antes de la migración de las specs de Axiom a identificadores funcionales (C-03-e):

1. El escáner de provisionales de la integración toma por provisionales reales los ejemplos literales de `tasks.md` y de las deltas, y bloquea el archivado.
2. La comprobación de colisiones de los ADDED con identificador definitivo compara por cadena y, con patrones con `<m>`, no ve que `RF-1` y `RF-001` son el mismo identificador.

## Problema y porqué

Fallo 1. `PlanIDs` (`internal/livespec/ids.go`, paso b) recorre con `reqid.ScanProvisional` el texto entero de cada delta; `changeArtifacts` (`internal/livespec/change.go`) toma como cita cualquier provisional de cualquier línea de los ficheros de tareas, y `docCitations` (`internal/livespec/odd_ids.go`) cualquier línea del documento ODD fuera de sus vallas. `Compose` (`internal/openspecdoc/compose.go`) rechaza con `ProvisionalIDError` cualquier token en el cuerpo de un requisito. Un ejemplo literal (en prosa, en código en línea o en un bloque de código) acaba como `provisional_malformed` o `provisional_undeclared`. En C-03-c hubo que reescribir los ejemplos con ordinales simbólicos (`prov:<serie>:<k>`) para poder archivar (Engram #2631 y #2638).

Fallo 2. `checkDefinitiveAdded` (`internal/livespec/ids.go:401`) busca el definitivo con `Registry.Lookup` (por cadena) y marca los repetidos del propio cambio con mapas por cadena. Con un patrón con `<m>` (ancho libre), un ADDED con `RF-1` no colisiona con un `RF-001` ya integrado aunque sean el mismo identificador según la decisión (c′) de C-03-c (duplicados por serie y número). `Registry.LookupIdentity` ya existe y no se usaba aquí.

## Alcance

- Escáner: cuentan solo los provisionales en posición de identificador, que son las referencias que el diseño de C-03-b reescribe (REQ-47.4):
  - la delta: títulos, cuerpos y preámbulo **fuera de código** (vallas y código en línea);
  - las líneas `Cubre:` de los ficheros de tareas y del documento ODD, fuera de vallas;
  - la columna `Requisito` de la matriz de `verify-report*.md` (sin cambios);
  - la valla `openspec` de `## Spec viva` del documento ODD, fuera de código en línea.
  El resto de los ficheros de tareas y del documento ODD ya no se valida, como `proposal.md` o `design.md`.
- `Compose` aplica la misma regla: un token dentro de código del cuerpo de un requisito es un ejemplo, no un provisional.
- Colisiones: identidad por serie y número en `checkDefinitiveAdded` (registro, repetidos del propio cambio, traza y `nextFree`).

Fuera de alcance:

- `IDMap.Substitute` no cambia: un token que coincide con un provisional declarado se sustituye también dentro de código (es una mención del mismo provisional; un ejemplo que no declara ningún ADDED no coincide con ninguna clave y queda intacto).
- Los bloques de código en línea que abarcan varias líneas: el escáner detecta el código en línea dentro de cada línea.

## Restricciones

- Cortex XDR borra ejecutables Go sueltos: sin `go build -o`, `go run` ni `go install`.
- Pruebas localizadas: `./internal/reqid`, `./internal/livespec`, `./internal/openspecdoc` y, en `./internal/cli`, solo con `-run`. Nunca la suite entera en local.
- El documento no contiene ningún provisional literal: el `axiom` del `PATH` (main, sin este arreglo) lo integra al cerrar.

## Decisiones

- **Qué es código.** Un bloque vallado (`` ``` `` o `~~~`, con las mismas reglas de apertura y cierre que `openspecdoc`) o un tramo de código en línea (una tirada de comillas invertidas cerrada por otra de la misma longitud en la misma línea). Una tirada sin cierre es literal.
- **Nueva función `reqid.ScanProvisionalRefs`.** `ScanProvisional` se conserva para los nombres (títulos) y para `isoevidence`; la nueva salta el código y la usan el plan (delta), `Compose` y la valla de la spec viva ODD.
- **Tareas y documento ODD.** Las citas salen de `isoevidence.ParseCubre`, la misma lectura de líneas `Cubre:` que usa `RewriteCubre` para reescribirlas.
- **Colisión (T2).** `checkDefinitiveAdded` usa `Registry.LookupIdentity` y claves `identityKey` en los mapas de repetidos, de traza y de ocupados; `nextFree` salta los números ocupados por identidad.

## TDD

Modo: desactivado (sin configuración en `axiom.yaml` ni petición). Cada tarea escribe primero la prueba que reproduce el fallo y se observa en rojo antes del arreglo. Ejecutor: `go test` por paquete, `-count=1`.

## Tareas

- [x] T1. Escáner: solo provisionales en posición de identificador y fuera de código. Ruta: inline (cambio localizado en `reqid`, `livespec` y `openspecdoc`).
  - Pruebas: `TestScanProvisionalRefsSkipsCode` (`reqid`: prosa, código en línea con tiradas simples y dobles, tirada sin cierre, vallas de comillas y de virgulillas, valla sin cerrar, cierre más corto); `TestPlanChangeIgnoresProvisionalExamplesInCodeAndTaskProse` (SDD: ejemplos en código en el cuerpo de un ADDED y en prosa y código de `tasks.md`; la spec viva conserva el ejemplo en código), `TestPlanChangeStillRejectsAnUndeclaredProvisionalInDeltaProse` (la prosa de la delta sigue validándose); `TestODDDocIgnoresProvisionalExamplesOutsideIdentifierPositions` y `TestODDDocRewriteRejectsAnUndeclaredProvisionalInACubreLine` (sustituye a la prueba que fijaba la validación del resto del documento). RED observado: `provisional_malformed` de la clave vacía en el ejemplo, en SDD y en ODD; `reqid.ScanProvisionalRefs` no existía.
  - Arreglo: `reqid.ScanProvisionalRefs` (salta vallas y código en línea); el paso b de `PlanIDs` y el guardián de `Compose` la usan; las citas de tareas y del documento ODD salen de `cubreCitations` (`isoevidence.ParseCubre`).
  - Evidencia: `go test -count=1 ./internal/reqid ./internal/livespec ./internal/openspecdoc ./internal/isoevidence` → `ok`; `go test -count=1 ./internal/cli -run 'ArchiveIntegrate|ArchiveCheck|ArchiveCompose|Provisional|SDDArchive|ODDIntegrat'` → `ok` (119 s); `gofmt -l` y `go vet` limpios.
- [x] T2. Colisión por identidad (serie y número) en `checkDefinitiveAdded`. Ruta: inline.
  - Pruebas: `TestPlanIDsDefinitiveAddedCollisionsCompareByIdentityWithFreeWidth` (`RF-<m>`: un ADDED con `RF-1` frente a `RF-001` de la canónica colisiona y el siguiente libre salta el `RF-02` del propio cambio, `RF-3`; `RF-5` y `RF-05` en dos capacidades del cambio colisionan) y el subcaso «repetido en otra capacidad con otro ancho» de `TestCheckArchivedTraceReportsADefinitiveThatAppearsMoreThanOnce` (`REQ-30.3` de la traza frente a `REQ-30.03` ajeno). RED observado: el plan aceptaba `RF-1` sin colisión y la comprobación de la traza archivada no veía el duplicado.
  - Arreglo: `checkDefinitiveAdded` compara con `Registry.LookupIdentity` y claves `identityKey` (traza, ocupados y repetidos); `nextFree` salta los ocupados por identidad; `resolveTraced` y `CheckArchivedTrace`, el mismo defecto, usan también `LookupIdentity`. `Registry.Lookup` queda sin uso en producción y pasa a `export_test.go` (la CI tiene un *ratchet* de código muerto).
  - Evidencia: `go test -count=1 ./internal/livespec ./internal/adoption/...` → `ok`; `go test -count=1 ./internal/cli -run 'ArchiveIntegrate|ArchiveCheck|ArchiveCompose|ArchiveAdopt|Provisional|SDDArchive|ODDIntegrat'` → `ok` (96 s); `go vet` de `livespec`, `cli`, `adoption` y `livingdoc` limpio.
- [x] T3. Cierre: spec viva, `axiom archive integrate --odd prov-ids-arreglo`, PR. Ruta: inline.
  - Spec viva: MODIFIED REQ-47.3 (regla 6) y REQ-47.4 (alcance de la validación, un escenario reescrito y dos nuevos) de `requirement-identifiers`. T2 no cambia requisitos: restaura la identidad por serie y número que ya fija la spec (decisión c′ de C-03-c).
  - Evidencia: `axiom archive integrate --odd prov-ids-arreglo --check` → `requirement-identifiers: composed`, sin escribir; después, la integración real en el mismo commit.

## Criterios de aceptación

- Prueba que reproduce cada fallo (rojo antes, verde después).
- `go test -count=1` en verde para `./internal/reqid`, `./internal/livespec`, `./internal/openspecdoc` y `./internal/cli` con `-run` sobre archive.
- Un commit por tarea.

## Spec viva

### Capacidad: requirement-identifiers

```openspec
## MODIFIED Requirements

### Requirement: Plan Único de Asignación sobre Todas las Capacidades, Validado antes de Escribir (REQ-47.3)

La asignación DEBE calcularse en un único plan para el cambio entero (todas las capacidades del cambio SDD, o todas las capacidades del documento ODD), antes de escribir ningún fichero, de modo que dos capacidades del mismo cambio nunca reciban el mismo número y el resultado no dependa de componer por dominio. El plan DEBE validar las reglas de la delta con identificadores mezclados y rechazar sin escribir, con un error tipado que nombre la capacidad, la sección, el requisito y el identificador:
(Previously: la regla 6 validaba la delta entera y los ficheros de tareas y el documento ODD enteros)

1. ADDED: admite un provisional o un identificador definitivo. Un ADDED con identificador definitivo DEBE seguir aceptándose con la comprobación de colisión vigente (REQ-30.4 de `odd-living-spec-integration`, REQ-29.3 de `sdd-iso-evidence`); el provisional NO es obligatorio.
2. MODIFIED y el nombre antiguo de un RENAMED, pendientes de integrar, cuyo nombre lleva un identificador válido según el patrón: DEBEN citar un identificador definitivo que exista, sin superar, en la especificación canónica de su capacidad. Si no existe, el error tipado es el de identificador inexistente. Un nombre sin identificador (solo admisible fuera de ISO; en ISO lo bloquea REQ-29.3) se resuelve por el título exacto, como fija REQ-30.22 de `sdd-archive-lifecycle`, y si no casa se rechaza con el motivo de composición vigente, no con el de identificador inexistente. El nombre nuevo de un RENAMED conserva el identificador del antiguo.
3. Lo ya integrado no está pendiente y no se rechaza (REQ-30.3 de `odd-living-spec-integration`): un REMOVED cuyo requisito ya no se resuelve en la canónica, un RENAMED cuyo nombre nuevo ya está y cuyo nombre antiguo ya no, y un MODIFIED cuyo requisito resuelto ya tiene el mismo cuerpo. `axiom sdd archive-compose`, que compone un único dominio sin clasificar lo ya integrado, sigue rechazando un REMOVED que no se resuelve con el error tipado de identificador inexistente (REQ-30.22).
4. Una capacidad sin especificación canónica cuya delta trae MODIFIED, REMOVED o RENAMED es un error tipado de capacidad inexistente, que nombra la capacidad, la sección y el requisito.
5. Un provisional en MODIFIED, REMOVED o RENAMED es un error tipado.
6. Un provisional citado sin un ADDED que lo declare en el mismo cambio es un error tipado, en las posiciones de identificador que REQ-47.4 somete a validación (la delta fuera de código, las líneas `Cubre:` de los ficheros de tareas y del documento ODD, y la columna del requisito de la matriz de verificación).
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

### Requirement: Sustitución por Token Completo y Reescritura Atómica del Provisional en Todos sus Destinos (REQ-47.4)

La sustitución de un provisional por su identificador definitivo DEBE hacerse por token completo: coincidencia exacta y sensible a mayúsculas, sin estar precedida ni seguida de una letra, una cifra, `_` o `-`, de modo que un provisional que es prefijo de otro nunca lo corrompe ni toca un texto que solo se le parece. La sustitución DEBE aplicarse a todas las apariciones del token en estos destinos:
(Previously: el resto del texto de los ficheros de tareas y del documento ODD se validaba, y un token dentro de código contaba como provisional)

1. La spec viva compuesta (títulos y cuerpos de los requisitos añadidos y, en una capacidad nueva, el texto entero que se escribe).
2. Las líneas `Cubre:` de los ficheros de tareas del cambio (`tasks.md` y `tasks.<rol>.md`), conservando la posición del token en la línea y el resto del texto byte a byte.
3. El documento ODD: la valla `openspec` de `## Spec viva` y sus líneas `Cubre:`, sin tocar el resto del documento.
4. La delta del cambio, entera, antes de archivarla.
5. La columna del requisito de las filas de la matriz de verificación (REQ-29.6 de `sdd-iso-evidence`) de los informes `verify-report*.md` del cambio, conservando el resto del informe byte a byte, de modo que un archivado reanudado no compare definitivos de la delta con provisionales de la matriz.

Alcance de la validación y de la reescritura fuera de esos destinos: solo se validan los provisionales en posición de identificador (REQ-47.3, reglas 6 y 9): los títulos, los cuerpos y el preámbulo de la delta, las líneas `Cubre:` de los ficheros de tareas y del documento ODD, la valla `openspec` de `## Spec viva` y la columna del requisito de la matriz. El resto del texto de los ficheros de tareas y del documento ODD, la sección del documento ODD que guarda el mapa, `proposal.md`, `design.md`, `explore.md` y los demás informes del cambio ni se validan ni se reescriben, de modo que pueden citar provisionales de ejemplo. Un token dentro de código (un bloque vallado o un tramo de código en línea) es un ejemplo de la forma, no un provisional: ni se valida ni impide la composición, y la spec viva puede conservarlo como ejemplo; si coincide con un provisional que declara un ADDED del cambio, se sustituye como cualquier otra aparición del token. «Declarado» significa declarado por un ADDED del cambio o por una entrada del mapa guardado. Fuera de código no existe un mecanismo de escape: un texto validado que necesita mostrar la forma del provisional en prosa usa un marcador que no es un token (por ejemplo, con la serie y el ordinal entre ángulos).

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

#### Scenario: La prosa de las tareas no se valida ni se reescribe

- **DADO** un `tasks.md` cuya descripción de una tarea menciona, fuera de una línea `Cubre:`, un provisional que ningún ADDED declara
- **CUANDO** se integra el cambio
- **ENTONCES** no se rechaza por esa mención
- **Y** la mención conserva sus bytes

#### Scenario: Un ejemplo en código de la delta no es un provisional

- **DADO** un ADDED cuyo cuerpo muestra, en código en línea y en un bloque vallado, un provisional que ningún ADDED declara
- **CUANDO** se integra el cambio
- **ENTONCES** no se rechaza por el ejemplo
- **Y** la spec viva conserva el ejemplo dentro del código

#### Scenario: Un provisional sin declarar en la prosa de la delta se rechaza

- **DADO** un ADDED cuyo cuerpo cita fuera de código un provisional que ningún ADDED ni el mapa guardado declaran
- **CUANDO** se planifica la asignación
- **ENTONCES** se rechaza con el error tipado de provisional no declarado y no se escribe nada

#### Scenario: La prosa de diseño no se valida

- **DADO** un `design.md` que cita como ejemplo un provisional que ningún ADDED declara
- **CUANDO** se planifica la asignación
- **ENTONCES** no se rechaza por esa cita
- **Y** `design.md` conserva sus bytes

#### Scenario: Un fallo al restaurar se nombra

- **DADO** un cambio con provisionales, un fallo inyectado al renombrar el último destino y un fallo al restaurar uno de los ya renombrados
- **CUANDO** se ejecuta la integración o el archivado
- **ENTONCES** finaliza con código distinto de cero nombrando el fichero que no pudo restaurar y la continuación con `git checkout -- <ruta>`
```

## Progreso

- T1 (`69eef05f`), T2 (`c033e20a`) y T3 (este commit) cerradas, un commit por tarea.
- Comprobaciones: `go test -count=1` de `./internal/reqid`, `./internal/livespec`, `./internal/openspecdoc`, `./internal/isoevidence` y `./internal/adoption/...` en `ok`; `./internal/cli` solo con `-run` sobre archivado e integración, en `ok`. La suite entera queda para la CI.

## Seguimientos

- `openspecdoc.ParseDelta` toma un `### Requirement:` dentro de un bloque vallado del cuerpo de un requisito como un requisito más de la delta (lo vio la primera versión de la prueba de T1); `Compose` sí respeta las vallas. No afecta a este arreglo, pero una spec que muestre un título de ejemplo dentro de una valla se integraría con un requisito de más.
