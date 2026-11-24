# Delta para sdd-archive-lifecycle

> Parte del estado vigente de `openspec/specs/sdd-archive-lifecycle/spec.md`. REQ-30.22 cambia la identidad del requisito al componer: MODIFIED, REMOVED y el nombre antiguo de RENAMED se resuelven por el identificador válido según el patrón del proyecto (REQ-44.4 de `requirement-identifiers`) cuando el nombre lo contiene, con vuelta al título exacto ante ambigüedad; ADDED y la colisión del nombre nuevo de RENAMED siguen por título. Toda composición hoy aceptada produce la misma salida byte a byte (REQ-30.23). El título del requisito no cambia.

## MODIFIED Requirements

### Requirement: Composición Nativa de OpenSpec con `axiom sdd archive-compose --canonical --delta` (REQ-30.22)

La composición de una delta sobre una especificación canónica y `axiom sdd archive-compose --canonical <ruta> --delta <ruta>` DEBEN producir la salida con identidad del requisito por su identificador cuando el nombre de la delta contiene un identificador válido según el patrón del proyecto (REQ-44.4 de `requirement-identifiers`; sin contexto de proyecto rige el patrón por defecto) y por el texto exacto del nombre si no lo contiene, orden de aplicación RENAMED, MODIFIED, REMOVED y ADDED, rechazo sin escribir ante una delta no aplicable y opciones `--supersede` y `--superseded-requirements`. El identificador de un nombre es el último identificador válido entre paréntesis.

**Resolución por identificador.** Se aplica a MODIFIED, REMOVED y al nombre antiguo de un RENAMED. Con un identificador en el nombre, se buscan los requisitos de la canónica que NO estén superados (los que no llevan el sufijo ` [SUPERSEDED / DEPRECADO]`) con ese identificador: si hay exactamente uno, es ese; si no hay ninguno, solo vale el requisito cuyo título coincide exactamente con el nombre (así se conserva la composición de hoy cuando la delta cita tal cual un requisito superado o un título con dos identificadores, REQ-30.23), y sin él la composición se rechaza como hoy (`no canonical requirement named "<nombre>"`); si hay varios, rige la coincidencia por título exacto cuando exactamente uno lo tiene, y si no la composición DEBE rechazarse sin escribir con un motivo tipado que nombre el identificador y los requisitos afectados (no se resuelve en silencio hacia uno de ellos). Sin identificador en el nombre, se busca el título exacto (primera coincidencia), como hasta ahora. Un MODIFIED cuyo título cambió pero conserva el identificador se aplica sobre el requisito que lleva ese identificador y lo sustituye por el bloque de la delta, que pasa a ser el nombre del requisito para las operaciones siguientes de la misma delta. Un REMOVED por identificador elimina (o, con `--supersede`, marca como superado) el requisito de la canónica resuelto, y la anotación usa el nombre que ese requisito tiene en la canónica. Un RENAMED sigue exigiendo ambos nombres (`Antiguo → Nuevo`) y la nota `(Reason: …)`. Los requisitos superados quedan excluidos de la resolución por identificador (su identificador está retirado y un MODIFIED no los reabre), pero conservan su `(ID)` en el título y siguen contando como identificador usado en el registro de la spec viva.

**Por título exacto, sin cambios.** ADDED (existencia previa), la colisión del nombre nuevo de un RENAMED y `--superseded-requirements` siguen emparejándose por título exacto: el compositor no rechaza un ADDED ni un nuevo nombre de RENAMED cuyo identificador ya figura bajo otro título; esa unicidad la imponen antes de componer REQ-29.3 (ISO) y REQ-30.4 (ODD).

**Archivado interrumpido (D17).** `axiom sdd archive-check` calcula «ya compuesto» por nombre para ADDED, MODIFIED y RENAMED, y por la misma resolución por identificador para REMOVED (un REMOVED se da por compuesto cuando ya no se resuelve en la canónica). La clasificación idempotente de la integración ODD (REQ-30.2) resuelve MODIFIED y REMOVED con esa misma resolución: un MODIFIED integrado deja la canónica con el nombre nuevo y una segunda ejecución informa `already`. Compositor, `archive-check` y clasificación ODD DEBEN coincidir siempre.

Toda composición aceptada antes de este cambio produce la misma salida byte a byte (REQ-30.23). `archive-compose` NO DEBE aceptar los modos `--document` ni `--in-place`: su uso DEBE rechazarse con un motivo tipado que indique que la baseline ajena se retiró y que la composición es `--canonical --delta`. (Es REQ-24.38 de `spec-baseline-integration`, aplicable a OpenSpec, más la retirada de los modos de documento.) Con un bloque `sdd:` inválido la composición en el contexto de un proyecto se niega sin escribir (REQ-44.9 de `requirement-identifiers`).
(Previously: la identidad era siempre el texto exacto del nombre del requisito, de modo que cambiar el título de un MODIFIED rompía la identidad; ahora es el identificador cuando el nombre lo contiene, con REMOVED y el nombre antiguo de RENAMED incluidos.)

Supuesto: se conservan sin cambios las pruebas nativas de composición de `--canonical --delta`.

#### Scenario: La salida nativa no cambia
- **DADO** una especificación canónica y un delta con ADDED, MODIFIED, REMOVED y RENAMED
- **CUANDO** se ejecuta `axiom sdd archive-compose --canonical <ruta> --delta <ruta>`
- **ENTONCES** la salida es idéntica byte a byte a la que producía la composición antes de este cambio

#### Scenario: Rechazo nativo sin escribir
- **DADO** un delta con un MODIFIED sobre un nombre de requisito inexistente
- **CUANDO** se ejecuta la composición con `--output <ruta>`
- **ENTONCES** finaliza con código distinto de cero nombrando la sección y el requisito
- **Y** no se escribe el fichero de salida

#### Scenario: `--supersede` conserva su comportamiento
- **DADO** un delta con un REMOVED y la opción `--supersede`
- **CUANDO** se compone
- **ENTONCES** el requisito se conserva marcado como superado, con su nota de motivo

#### Scenario: Los modos de documento se rechazan
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom sdd archive-compose --document <dominio> --delta <ruta>` o con `--in-place`
- **ENTONCES** finaliza con código distinto de cero con un motivo tipado que remite a `--canonical --delta`
- **Y** no se escribe ningún fichero

#### Scenario: Un MODIFIED con el título cambiado conserva el identificador
- **DADO** una canónica con `### Requirement: Cobro (REQ-30.1)` y una delta con un MODIFIED `### Requirement: Cobro con IVA (REQ-30.1)`
- **CUANDO** se compone
- **ENTONCES** la salida contiene el bloque de la delta con el título `Cobro con IVA (REQ-30.1)` en la posición del requisito original
- **Y** no queda ningún requisito `Cobro (REQ-30.1)`
- **Y** no se rechaza la composición

#### Scenario: Sin identificador en el título se empareja por el texto exacto
- **DADO** una canónica con `### Requirement: Cobro` y una delta con un MODIFIED `### Requirement: Cobro`
- **CUANDO** se compone
- **ENTONCES** el MODIFIED se aplica por coincidencia exacta del nombre, como antes de este cambio

#### Scenario: Un MODIFIED sin identificador cuyo nombre no coincide se rechaza
- **DADO** una canónica con `### Requirement: Cobro` y una delta con un MODIFIED `### Requirement: Cobro con IVA`
- **CUANDO** se compone
- **ENTONCES** finaliza con código distinto de cero nombrando la sección MODIFIED y el requisito
- **Y** no se escribe el fichero de salida

#### Scenario: Con un patrón declarado se empareja por el identificador de ese patrón
- **DADO** un proyecto con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`, una canónica con `### Requirement: Alta de usuario (RF-USU-01)` y una delta con un MODIFIED `### Requirement: Alta de usuario con correo (RF-USU-01)`
- **CUANDO** se compone en el contexto de ese proyecto
- **ENTONCES** el MODIFIED sustituye al requisito `RF-USU-01`

#### Scenario: RENAMED sigue exigiendo ambos nombres
- **DADO** una delta con un RENAMED `Cobro (REQ-30.1) → Cobro con IVA (REQ-30.1)` sobre una canónica que contiene `Cobro (REQ-30.1)`
- **CUANDO** se compone
- **ENTONCES** el requisito pasa a llamarse `Cobro con IVA (REQ-30.1)`
- **Y** un RENAMED al que le falte uno de los dos nombres se rechaza sin escribir

#### Scenario: Un MODIFIED sobre el nombre nuevo de un RENAMED de la misma delta se aplica
- **DADO** una delta con un RENAMED `Cobro (REQ-30.1) → Cobro con IVA (REQ-30.1)` y un MODIFIED `Cobro con IVA (REQ-30.1)`
- **CUANDO** se compone
- **ENTONCES** se aplican primero el RENAMED y después el MODIFIED, con el cuerpo del MODIFIED en el requisito renombrado

#### Scenario: Un identificador repetido en la canónica no se resuelve en silencio
- **DADO** una canónica con dos requisitos `Cobro (REQ-30.1)` y `Devolución (REQ-30.1)` y una delta con un MODIFIED `Reembolso (REQ-30.1)`, cuyo nombre no coincide con ninguno de los dos
- **CUANDO** se compone
- **ENTONCES** finaliza con código distinto de cero con un motivo tipado que nombra `REQ-30.1` y los dos requisitos
- **Y** no se escribe el fichero de salida

#### Scenario: Con un identificador repetido, la coincidencia exacta de nombre sigue valiendo
- **DADO** una canónica con dos requisitos `Cobro (REQ-30.1)` y `Devolución (REQ-30.1)` y una delta con un MODIFIED `Cobro (REQ-30.1)`, que coincide por nombre exacto con exactamente uno
- **CUANDO** se compone
- **ENTONCES** el MODIFIED se aplica sobre `Cobro (REQ-30.1)`, como antes de este cambio

#### Scenario: Una composición hoy aceptada produce la misma salida
- **DADO** el conjunto de composiciones que el compositor aceptaba antes de este cambio
- **CUANDO** se componen con el compositor nuevo
- **ENTONCES** cada salida es idéntica byte a byte a la anterior (REQ-30.23)

#### Scenario: Un REMOVED con el título obsoleto se resuelve por identificador
- **DADO** una canónica con `### Requirement: Cobro con IVA (REQ-30.1)` y una delta con un REMOVED `### Requirement: Cobro (REQ-30.1)`
- **CUANDO** se compone, con y sin `--supersede`
- **ENTONCES** se elimina (o, con `--supersede`, se marca como superado) el requisito `Cobro con IVA (REQ-30.1)`
- **Y** con `--supersede` la anotación de superado usa el nombre `Cobro con IVA (REQ-30.1)` de la canónica

#### Scenario: Un RENAMED resuelve su nombre antiguo por identificador
- **DADO** una canónica con `### Requirement: Cobro (REQ-30.1)` y una delta con un RENAMED `Cobro antiguo (REQ-30.1) → Cobro con IVA (REQ-30.1)` con su nota `(Reason: …)`
- **CUANDO** se compone
- **ENTONCES** el requisito `Cobro (REQ-30.1)` pasa a llamarse `Cobro con IVA (REQ-30.1)`

#### Scenario: Un MODIFIED que cambia el identificador no se resuelve
- **DADO** una canónica con `### Requirement: Cobro (REQ-30.1)` y una delta con un MODIFIED `### Requirement: Cobro (REQ-30.2)`, sin ningún requisito con `REQ-30.2`
- **CUANDO** se compone
- **ENTONCES** finaliza con código distinto de cero nombrando la sección MODIFIED y el requisito
- **Y** no se escribe el fichero de salida

#### Scenario: Los requisitos superados quedan fuera de la resolución por identificador
- **DADO** una canónica con `### Requirement: Cobro (REQ-30.1) [SUPERSEDED / DEPRECADO]` y una delta con un MODIFIED `### Requirement: Cobro con IVA (REQ-30.1)`
- **CUANDO** se compone
- **ENTONCES** se rechaza como requisito inexistente sin escribir
- **Y** si la canónica tuviera además un requisito vivo `Cobro nuevo (REQ-30.1)`, el MODIFIED se aplicaría sobre ese

#### Scenario: ADDED y el nombre nuevo de un RENAMED se emparejan por título
- **DADO** una canónica con `### Requirement: Cobro (REQ-30.1)` y una delta con un ADDED `### Requirement: Devolución (REQ-30.1)`
- **CUANDO** se compone
- **ENTONCES** el ADDED se añade, porque no existe ningún requisito con ese título exacto
- **Y** el compositor no valida la unicidad del identificador (la imponen REQ-29.3 y REQ-30.4 antes de componer)

#### Scenario: Con un patrón declarado, un requisito con identificador de otro patrón se empareja por título
- **DADO** un proyecto con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`, una canónica con `### Requirement: Cobro (REQ-30.1)` y una delta con un MODIFIED `### Requirement: Cobro (REQ-30.1)`
- **CUANDO** se compone en el contexto de ese proyecto
- **ENTONCES** como `REQ-30.1` no es un identificador válido, el MODIFIED se empareja por título exacto y se aplica

#### Scenario: `archive-check` con un REMOVED por identificador y título obsoleto
- **DADO** una delta con un REMOVED `Cobro (REQ-30.1)` y una canónica que aún contiene `Cobro con IVA (REQ-30.1)`
- **CUANDO** se ejecuta `axiom sdd archive-check` antes de componer
- **ENTONCES** el dominio no figura como ya compuesto
- **Y** tras componer y volver a consultar, el dominio figura como ya compuesto

#### Scenario: Una segunda integración ODD de un MODIFIED por identificador es idempotente
- **DADO** una delta ODD con un MODIFIED `Cobro con IVA (REQ-30.1)` ya integrado sobre `Cobro (REQ-30.1)`
- **CUANDO** se repite la integración
- **ENTONCES** el MODIFIED se clasifica como ya aplicado y no se escribe nada

#### Scenario: Con un bloque `sdd:` inválido la composición se niega
- **DADO** un proyecto cuyo `axiom.yaml` tiene un bloque `sdd:` inválido y una delta de un cambio de ese proyecto
- **CUANDO** se ejecuta `axiom sdd archive-compose` en su contexto
- **ENTONCES** finaliza con código distinto de cero con la continuación `axiom workspace validate`
- **Y** no se escribe ningún fichero
