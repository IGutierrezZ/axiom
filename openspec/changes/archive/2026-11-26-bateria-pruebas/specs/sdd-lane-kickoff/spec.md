# Delta para Carril SDD y Kickoff (sdd-lane-kickoff)

> **Cambio:** `bateria-pruebas` (C-06-c). **Idioma:** castellano de España; identificadores, claves y tokens en inglés.
> RF-KOF-01 a RF-KOF-16 no cambian de texto. La declaración del bloque `acceptance_tests` vive en `workspace-topology` (RF-TOP-19, RF-TOP-20); el digest y la parada de la compuerta `tasks`, en `sdd-block-review-gates` (RF-GAT-31, RF-GAT-32); la pestaña, en `local-web-dashboard` (RF-WEB-17). El kickoff NO sella `acceptance_tests`: la activación se lee siempre del estado vivo de `axiom.yaml`.

## ADDED Requirements

### Requirement: `sdd-tasks` Genera `pruebas.md` solo en SDD, con la Función Activa y Después del Design (RF-KOF-17)

La fase `sdd-tasks` DEBE escribir `pruebas.md` en la carpeta del cambio si y solo si se cumplen a la vez: el cambio va por el carril SDD, y `axiom sdd status --json` proyecta `acceptanceTests` activo (RF-KOF-22), esto es, `acceptance_tests.enabled: true` en `axiom.yaml`. Con `enabled: false`, sin el bloque o con un bloque inválido, NO DEBE generar nada, y el digest de `tasks`, el status y el resto del flujo NO DEBEN cambiar. Un cambio ODD NUNCA genera `pruebas.md`. El carácter `continuous` o `checkpointed` del cambio no altera la regla.

Al generarlo, `sdd-tasks` DEBE minimizar el número de casos aplicando, por este orden: (1) trazado obligatorio, esto es, cada requisito y cada escenario ADDED o MODIFIED queda citado; (2) particiones de equivalencia y valores límite; (3) pairwise solo con tres o más parámetros independientes; (4) fusión de casos con las mismas precondiciones. DEBE ejecutar después `axiom sdd tests check` (RF-KOF-21) y corregir el fichero hasta que no queden hallazgos o dejar constancia de los que no pueda resolver. La minimización es responsabilidad del modelo: Go solo comprueba e informa. En la parada `tasks` presenta `pruebas.md` según RF-GAT-32.

`sdd-apply` NO DEBE ampliar el alcance de `pruebas.md`: solo PUEDE refinar rutas o URL en `Pasos`, sin añadir, quitar ni reasignar casos ni alterar `Cubre`. La doctrina común DEBE vivir en `_shared/sdd-acceptance-tests.md`, enlazada desde la skill `sdd-tasks` y desde sus agentes delgados (claude, cursor y copilot), y un test de pines DEBE fijar: la condición SDD y `acceptanceTests` activo, que ODD nunca genera, las cuatro capas, la ejecución de `axiom sdd tests check` y la regla de refinado de `sdd-apply`.

#### Scenario: Función activa en SDD
- **DADO** un fixture `multirepo` con `acceptance_tests.enabled: true`, un cambio SDD tras design y el status con `acceptanceTests` activo
- **CUANDO** se ejecuta la fase `sdd-tasks`
- **ENTONCES** la doctrina exige escribir `pruebas.md` con IDs `P-NN` y ejecutar `axiom sdd tests check`

#### Scenario: Función no activa no genera nada
- **DADO** un proyecto con `enabled: false` o sin el bloque
- **CUANDO** se ejecuta la fase `sdd-tasks`
- **ENTONCES** no se genera `pruebas.md`
- **Y** el status no proyecta aviso y el digest de `tasks` no cambia

#### Scenario: ODD nunca genera
- **DADO** un cambio por el carril ODD en un proyecto con la función activa
- **CUANDO** se ejecuta el flujo ODD
- **ENTONCES** no se genera `pruebas.md`
- **Y** la doctrina lo condiciona a SDD, lo que fija un test de pines

#### Scenario: `sdd-apply` solo refina rutas o URL
- **DADO** un `pruebas.md` aprobado
- **CUANDO** `sdd-apply` lo refina
- **ENTONCES** solo modifica rutas o URL de `Pasos`
- **Y** no añade casos ni cambia `Cubre`, según `_shared/sdd-acceptance-tests.md`

### Requirement: Gramática Cerrada de la Tabla de `pruebas.md` (RF-KOF-18)

`pruebas.md` DEBE contener una única tabla markdown cuyo parseo sea determinista, con estas reglas:

- **Cabecera.** Las nueve columnas, con estos nombres exactos y en castellano: `ID`, `Cubre`, `Rol`, `Tipo`, `Prioridad`, `Entornos`, `Precondiciones`, `Pasos`, `Resultado esperado`. Se localizan por nombre (tras recortar espacios, sensible a mayúsculas), en cualquier orden; una columna ausente, duplicada o con otro nombre hace la forma de la tabla inválida. Tras la cabecera va la línea separadora de markdown. Cada fila DEBE tener tantas celdas como columnas. Un `|` dentro de una celda se escribe `\|`. Todo texto fuera de la tabla (títulos, notas) se ignora.
- **ID.** `P-` seguido de al menos dos dígitos (`P-01`, `P-12`, `P-100`). DEBE ser único en el fichero y estable: no se renumera al editar, de modo que un caso conserva su ID mientras exista. Un ID mal formado o repetido es un hallazgo.
- **Tipo.** Exactamente `manual` o `automatizable`.
- **Prioridad.** Exactamente `alta`, `media` o `baja`.
- **Entornos.** Lista separada por `;` (con espacios opcionales alrededor), no vacía, sin repetidos, cuyos elementos pertenezcan, con comparación exacta, a `acceptance_tests.environments` del proyecto.
- **Rol.** Un único identificador que sea un rol sellado del cambio en `kickoff.yaml` (en multirepo el sellado es siempre `checkpointed`).
- **Cubre.** Según RF-KOF-19.
- **Precondiciones.** Texto libre; PUEDE estar vacío.
- **Pasos** y **Resultado esperado.** Texto libre no vacío.

Una celda con un valor fuera de su dominio, o una celda obligatoria vacía, es un hallazgo con el número de línea de la fila. La ausencia de tabla o de cabecera es el hallazgo de forma de tabla inválida.

#### Scenario: Tabla válida
- **DADO** un `pruebas.md` con las nueve columnas, IDs `P-01` y `P-02`, `Tipo` `manual`, `Prioridad` `alta`, `Entornos` `dev; pre` declarados y un `Rol` sellado
- **CUANDO** se ejecuta `axiom sdd tests check`
- **ENTONCES** la tabla se parsea sin hallazgos de forma

#### Scenario: Columnas en otro orden
- **DADO** una tabla con las nueve columnas en un orden distinto
- **CUANDO** se parsea
- **ENTONCES** cada celda se asocia a su columna por nombre

#### Scenario: Cabecera inválida
- **DADO** una tabla sin la columna `Cubre`, con una columna extra o con `Prioridad` escrita `prioridad`
- **CUANDO** se parsea
- **ENTONCES** se informa de la forma de tabla inválida con la columna responsable

#### Scenario: ID mal formado o duplicado
- **DADO** una fila con ID `P1` o `p-01`, y dos filas con el ID `P-03`
- **CUANDO** se parsea
- **ENTONCES** cada caso produce un hallazgo con la línea de la fila

#### Scenario: Tipo o prioridad fuera de dominio
- **DADO** una fila con `Tipo` `exploratoria` y otra con `Prioridad` `urgente`
- **CUANDO** se comprueba
- **ENTONCES** cada una produce un hallazgo con su valor y sus valores admitidos

#### Scenario: Entorno no declarado o repetido
- **DADO** `environments: [dev, pre]` y una fila con `Entornos` `prod`, otra con `dev; dev` y otra vacía
- **CUANDO** se comprueba
- **ENTONCES** cada fila produce un hallazgo de entorno inválido

#### Scenario: Rol no sellado
- **DADO** un cambio con el rol único `fullstack` y una fila con `Rol` `backend`
- **CUANDO** se comprueba
- **ENTONCES** produce un hallazgo de rol inválido que lista los roles sellados

#### Scenario: Barra escapada en una celda
- **DADO** una celda de `Pasos` con `a \| b`
- **CUANDO** se parsea
- **ENTONCES** la celda contiene `a | b` y la fila conserva sus nueve celdas

### Requirement: Sintaxis de `Cubre` y Reconocimiento de IDs solo por el Puerto `reqid` (RF-KOF-19)

La columna `Cubre` DEBE ser una lista no vacía separada por `;`, porque los nombres de escenario pueden contener comas. Cada entrada, recortada de espacios, es de una de estas dos formas:

- `ID`: cita un requisito;
- `ID#<nombre exacto del escenario>`: cita ese escenario de ese requisito. El separador es el primer `#`; el nombre es el texto del encabezado `#### Scenario:` recortado de espacios y con comparación exacta, sensible a mayúsculas.

Un ID SOLO se reconoce con el puerto `internal/reqid`: `REQ-<n>.<m>`, `RF-<DOM>-<NN>`, los identificadores provisionales y el `sdd.id_pattern` del proyecto. No hay ningún reconocedor adicional. Una entrada cuyo ID no reconoce `reqid`, o que reconociéndolo no figura en la delta del cambio, es un hallazgo de ID desconocido; una entrada `ID#nombre` cuyo requisito existe pero no tiene ese escenario es un hallazgo de escenario desconocido; una celda `Cubre` vacía o con una entrada vacía es el hallazgo de caso sin cubre. Un escenario solo queda cubierto si se cita explícitamente con `ID#nombre`; un requisito queda cubierto por su ID suelto o por cualquiera de sus escenarios citados. Un nombre de escenario que contenga `;` no puede citarse: es un hallazgo de escenario no citable.

#### Scenario: Requisito citado por su ID suelto
- **DADO** una fila con `Cubre` `RF-KOF-17`
- **CUANDO** se calcula la cobertura
- **ENTONCES** el requisito queda cubierto
- **Y** ninguno de sus escenarios queda cubierto por esa entrada

#### Scenario: Escenario citado con `ID#nombre`
- **DADO** una fila con `Cubre` `RF-KOF-17#Función activa en SDD; RF-KOF-17#ODD nunca genera`
- **CUANDO** se calcula la cobertura
- **ENTONCES** ambos escenarios y el requisito quedan cubiertos

#### Scenario: Nombre de escenario con comas
- **DADO** un escenario cuyo nombre contiene comas y una entrada `RF-XXX-01#nombre, con comas`
- **CUANDO** se calcula la cobertura
- **ENTONCES** la entrada se interpreta como una sola cita

#### Scenario: ID no reconocido por `reqid`
- **DADO** una entrada `ABC-9` que `reqid` no reconoce
- **CUANDO** se comprueba
- **ENTONCES** produce un hallazgo de ID desconocido

#### Scenario: ID o escenario inexistente en la delta
- **DADO** una entrada con un ID válido ausente de la delta, y otra `ID#nombre` con un nombre inexistente
- **CUANDO** se comprueba
- **ENTONCES** la primera produce un hallazgo de ID desconocido y la segunda uno de escenario desconocido

#### Scenario: Caso sin cubre
- **DADO** una fila con `Cubre` vacío o con `RF-X-01;;RF-X-02`
- **CUANDO** se comprueba
- **ENTONCES** produce un hallazgo de caso sin cubre con la línea de la fila

#### Scenario: Identificador provisional y patrón del proyecto
- **DADO** un requisito con ID provisional y otro con un ID del `sdd.id_pattern` del proyecto
- **CUANDO** se citan en `Cubre`
- **ENTONCES** `reqid` los reconoce y quedan cubiertos

### Requirement: `acceptancetests.Check` Cruza `pruebas.md` con las Specs Delta y Define los Hallazgos y el Informe (RF-KOF-20)

`internal/acceptancetests.Check` DEBE parsear `pruebas.md` y cruzarlo con los requisitos y escenarios de los bloques ADDED y MODIFIED de las specs delta del cambio (`specs/<capacidad>/spec.md`), usando `Scenarios` y `AffectedFromDelta` de `internal/isoevidence`. Los bloques REMOVED y RENAMED quedan fuera de la base de cobertura. Es la única implementación del cálculo, compartida por el verbo, el status y el dashboard. Es de solo lectura y determinista: la misma entrada da el mismo informe, con los hallazgos ordenados por fichero, línea y código.

Son hallazgos que hacen fallar la comprobación, con código estable:

| Código | Condición |
|--------|-----------|
| `pruebas_ausente` | La función está activa, hay `tasks` y falta `pruebas.md` |
| `tabla_invalida` | Tabla o cabecera ausente o con forma no válida (RF-KOF-18) |
| `celda_invalida` | Celda obligatoria vacía o valor fuera de dominio (`Tipo`, `Prioridad`) |
| `id_caso_invalido` | `P-NN` mal formado |
| `id_caso_duplicado` | `P-NN` repetido |
| `caso_sin_cubre` | `Cubre` vacío o con una entrada vacía |
| `entorno_invalido` | Entorno ausente, repetido o fuera de `acceptance_tests.environments` |
| `rol_invalido` | Rol que no es un rol sellado del cambio |
| `id_desconocido` | ID no reconocido por `reqid` o ausente de la delta |
| `escenario_desconocido` | `ID#nombre` cuyo requisito no tiene ese escenario |
| `requisito_sin_id` | Requisito ADDED o MODIFIED sin ID citable |
| `id_duplicado_en_delta` | El mismo ID figura en dos requisitos ADDED o MODIFIED de la delta |
| `escenario_ambiguo` | Nombre de escenario repetido dentro de un mismo requisito, o que contiene `;` |
| `requisito_sin_cubrir` | Requisito con ID que ningún caso cita |
| `escenario_sin_cubrir` | Escenario que ningún caso cita con `ID#nombre` |

El informe DEBE contener: los escenarios cubiertos sobre el total (`M/M`), los requisitos cubiertos sobre el total (`M/M`, donde el total incluye los requisitos sin ID, que no pueden cubrirse), el número de casos y la lista de hallazgos con código, línea y mensaje. Sin hallazgos, el informe declara cobertura completa. El Check NO DEBE modificar ningún fichero.

#### Scenario: Cobertura completa
- **DADO** una delta con 2 requisitos y 5 escenarios, y un `pruebas.md` válido con 4 casos que citan todos con `ID#nombre`
- **CUANDO** se ejecuta `Check`
- **ENTONCES** el informe declara escenarios 5/5, requisitos 2/2, 4 casos y ningún hallazgo

#### Scenario: Escenario sin cubrir
- **DADO** un requisito citado solo por su ID suelto y con 2 escenarios
- **CUANDO** se ejecuta `Check`
- **ENTONCES** informa de `escenario_sin_cubrir` por cada escenario
- **Y** el requisito cuenta como cubierto

#### Scenario: Requisito sin cubrir
- **DADO** un requisito ADDED que ningún caso cita
- **CUANDO** se ejecuta `Check`
- **ENTONCES** informa de `requisito_sin_cubrir` y de `escenario_sin_cubrir` por cada uno de sus escenarios

#### Scenario: Requisito sin ID
- **DADO** un requisito ADDED o MODIFIED sin ID reconocible por `reqid`
- **CUANDO** se ejecuta `Check`
- **ENTONCES** informa de `requisito_sin_id` y no lo ignora en silencio
- **Y** cuenta en el total de requisitos del informe

#### Scenario: Escenario ambiguo
- **DADO** un requisito con dos escenarios del mismo nombre
- **CUANDO** se ejecuta `Check`
- **ENTONCES** informa de `escenario_ambiguo` con el nombre repetido

#### Scenario: MODIFIED cuenta y REMOVED no
- **DADO** una delta con un requisito MODIFIED, otro REMOVED y otro RENAMED
- **CUANDO** se ejecuta `Check`
- **ENTONCES** solo el MODIFIED entra en la base de cobertura
- **Y** citar el REMOVED produce `id_desconocido`

#### Scenario: Un hallazgo por cada causa
- **DADO** casos con un ID desconocido, un escenario desconocido, un entorno no declarado, un rol no sellado, un tipo fuera de dominio, una prioridad fuera de dominio y un `P-NN` duplicado
- **CUANDO** se ejecuta `Check`
- **ENTONCES** informa de `id_desconocido`, `escenario_desconocido`, `entorno_invalido`, `rol_invalido`, `celda_invalida` (dos veces) e `id_caso_duplicado`

#### Scenario: Determinismo y solo lectura
- **DADO** el mismo cambio
- **CUANDO** se ejecuta `Check` dos veces
- **ENTONCES** los informes son idénticos
- **Y** ningún fichero del cambio se modifica

### Requirement: Verbo de Solo Lectura `axiom sdd tests check` (RF-KOF-21)

`axiom sdd tests check --cwd <repo> --change <c> [--json]` DEBE ejecutar `acceptancetests.Check` y presentar el informe de RF-KOF-20. Es de solo lectura, vive bajo un subdespachador `tests` (como `gate`) y se registra en `cmd/axiom/main.go`, en su ayuda y en `main_test.go`. En texto DEBE imprimir los escenarios M/M, los requisitos M/M, el número de casos y cada hallazgo con código, línea y mensaje; con `--json` DEBE emitir un objeto con `active`, `environments`, `present`, `cases`, `scenarios {covered, total}`, `requirements {covered, total}` y `findings [{code, line, message}]`. El objeto PUEDE llevar claves adicionales de identificación (por ejemplo `schema`, `change`, `file` o `error`); un consumidor NO DEBE rechazarlas.

El verbo DEBE terminar con éxito si no hay hallazgos y con error (código de salida distinto de cero) si hay algún hallazgo, o si la función está activa y falta `pruebas.md` después de `tasks` (`pruebas_ausente`). Con la función no activa y sin `pruebas.md`, DEBE terminar con éxito indicando que las pruebas de aceptación no están activadas, sin hallazgos. Con la función no activa y con `pruebas.md` presente, comprueba el fichero igualmente. Un cambio inexistente o sin specs delta es un error de uso, no un informe.

#### Scenario: Informe en texto y JSON
- **DADO** un cambio con `pruebas.md` válido
- **CUANDO** se ejecuta `axiom sdd tests check --cwd <repo> --change <c>` y con `--json`
- **ENTONCES** ambos salen con código 0 y muestran los mismos escenarios M/M, requisitos M/M y número de casos

#### Scenario: Con hallazgos sale en error
- **DADO** un `pruebas.md` con un escenario sin cubrir
- **CUANDO** se ejecuta el verbo
- **ENTONCES** sale con código distinto de cero e imprime `escenario_sin_cubrir` con su línea

#### Scenario: Activa y falta `pruebas.md`
- **DADO** la función activa, `tasks` terminado y sin `pruebas.md`
- **CUANDO** se ejecuta el verbo
- **ENTONCES** sale con error y `pruebas_ausente`

#### Scenario: No activa y sin fichero
- **DADO** `enabled: false` o sin bloque y sin `pruebas.md`
- **CUANDO** se ejecuta el verbo
- **ENTONCES** sale con código 0 e indica que las pruebas de aceptación no están activadas

#### Scenario: Registro del verbo
- **CUANDO** se ejecutan `axiom help` y las pruebas de `main_test.go`
- **ENTONCES** la ayuda lista `sdd tests check` y la prueba falla si se elimina del despachador

### Requirement: El Status Proyecta `acceptanceTests` de Forma Aditiva y Avisa sin Bloquear la Ausencia de `pruebas.md` (RF-KOF-22)

`axiom sdd status --json` DEBE proyectar la clave aditiva `acceptanceTests` con el estado vivo de `axiom.yaml`: la activación y los entornos, o el error del bloque si es inválido (sin asumir un valor), y, si existe `pruebas.md`, el resultado de `acceptancetests.Check`: número de casos, escenarios y requisitos cubiertos (M/M) y número de hallazgos. Si la función está activa, `tasks` está terminado y falta `pruebas.md`, DEBE emitir un aviso informativo. Con `enabled: false` o sin el bloque, DEBE proyectar la función como no activa y NO DEBE emitir ningún aviso; omitir la clave `acceptanceTests` equivale a proyectarla como no activa, para que el JSON del status de un proyecto sin el bloque no cambie.

La proyección y el aviso NO DEBEN alterar `nextRecommended`, las dependencias, los motivos de bloqueo ni los digests, y NO DEBEN bloquear la compuerta ni el avance. Ninguna clave existente del status DEBE cambiar de nombre, tipo ni valor. El contrato publicado (`_shared/sdd-status-contract.md`) DEBE documentar la clave.

#### Scenario: Función activa con `pruebas.md`
- **DADO** la función activa y un `pruebas.md` con 2 hallazgos
- **CUANDO** se ejecuta `axiom sdd status --change <c> --json`
- **ENTONCES** `acceptanceTests` proyecta activación, entornos, cobertura y 2 hallazgos

#### Scenario: Aviso por ausencia sin bloqueo
- **DADO** la función activa, `tasks` terminado y sin `pruebas.md`
- **CUANDO** se consulta el status
- **ENTONCES** `acceptanceTests` incluye un aviso
- **Y** `nextRecommended` y las dependencias son idénticos a los del mismo cambio con `pruebas.md` ausente y la función no activa

#### Scenario: Función no activa
- **DADO** `enabled: false` o sin el bloque
- **CUANDO** se consulta el status
- **ENTONCES** no hay aviso y los digests no cambian

#### Scenario: Declaración inválida
- **DADO** un bloque `acceptance_tests` inválido
- **CUANDO** se consulta el status
- **ENTONCES** `acceptanceTests` proyecta el error del bloque
- **Y** no se asume activación

### Requirement: El Archivado Reescribe los Provisionales de la Columna `Cubre` de `pruebas.md` (RF-KOF-23)

Al archivar, `internal/livespec` DEBE reescribir los IDs provisionales a su ID definitivo también en la columna `Cubre` de `pruebas.md`, como un tercer tipo de fichero del cambio (junto a los dos existentes), con la misma tabla de correspondencia que para el resto de ficheros. La reescritura DEBE tocar solo el ID de cada entrada de `Cubre` (incluida la forma `ID#nombre`, donde el nombre del escenario no cambia), conservar el resto del fichero byte a byte, y ser idempotente. Sin `pruebas.md`, DEBE ser una operación nula. Un ID de `Cubre` que no es provisional no se toca.

#### Scenario: Provisionales reescritos en `Cubre`
- **DADO** un `pruebas.md` con `Cubre` `RF-PROV-1#Escenario A; RF-PROV-1`
- **CUANDO** se archiva el cambio y `RF-PROV-1` se integra como `RF-KOF-30`
- **ENTONCES** `Cubre` queda `RF-KOF-30#Escenario A; RF-KOF-30`

#### Scenario: El resto del fichero no cambia
- **DADO** el mismo `pruebas.md`
- **CUANDO** se archiva
- **ENTONCES** las demás columnas y el texto fuera de la tabla no cambian

#### Scenario: Idempotencia y ausencia
- **DADO** un `pruebas.md` ya reescrito, y un cambio sin `pruebas.md`
- **CUANDO** se vuelve a archivar o se archiva
- **ENTONCES** no hay cambios y no se produce error

#### Scenario: IDs definitivos intactos
- **DADO** una entrada `Cubre` con un ID no provisional
- **CUANDO** se archiva
- **ENTONCES** la entrada no cambia
