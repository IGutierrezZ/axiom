# Especificación de Requerimientos: Integración del Carril ODD en la Spec Viva de OpenSpec (REQ-30.1 a REQ-30.8)

> **Cambio:** `spec-viva-openspec` · **Tipo:** capacidad nueva (especificación completa, no delta)
> **Idioma:** Español (castellano peninsular)

## Propósito

Definir cómo un incremento o un bug tramitado por el carril ODD integra sus requisitos en la spec viva única del proyecto (`openspec/specs/`) sin pasar por el carril SDD: el formato de las secciones `## Spec viva` e `## Impacto en la spec` del documento `odd/tasks/<feature>.md`, el verbo `axiom archive integrate --odd <feature>`, la comprobación determinista de colisión de identificadores, la obligación de declarar el impacto en la spec de todo bug ODD y la comprobación estructural del contexto técnico.

**Resultado esperado:** al cerrar una feature ODD, su delta OpenSpec queda integrada en `openspec/specs/` con un verbo idempotente y sin compuertas; todo bug ODD declara su impacto en la spec, aunque sea «Sin impacto en la spec» con motivo; y nunca se modifica la documentación de referencia.

## Términos

| Término | Definición |
|---|---|
| Documento ODD | Fichero `odd/tasks/<feature>.md` de la rama actual: el único artefacto de una feature ODD. |
| Sección `## Spec viva` | Sección del documento ODD que contiene, por capacidad, la delta OpenSpec que el incremento aplica a `openspec/specs/`. |
| Marcador de «sin impacto» | Sección `## Impacto en la spec` con la línea literal `Sin impacto en la spec` y una línea `Motivo: <una línea>` no vacía; mismo contrato en ODD y SDD. |
| Bug ODD | Documento ODD cuya cabecera contiene la línea `Tipo: bug`. |
| Spec viva | `openspec/specs/<capacidad>/spec.md`: el único formato de especificación viva del proyecto. |
| Delta activa | Delta de spec de un cambio SDD vigente bajo `openspec/changes/<cambio>/specs/`, excluida la carpeta `archive/`. |

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| REQ-30.1 | Sección `## Spec viva` del documento ODD | Un bloque vallado por capacidad con la delta OpenSpec |
| REQ-30.2 | `axiom archive integrate --odd <feature>` | Compone sobre `openspec/specs/`, todo o nada, sin compuertas |
| REQ-30.3 | Idempotencia de la integración | Repetirla no cambia nada y termina con éxito |
| REQ-30.4 | Colisión de identificadores | Un `REQ-n.m` ADDED ya usado se rechaza con el siguiente número libre |
| REQ-30.5 | Marcador `Sin impacto en la spec` y bugs ODD | Mismo contrato que el carril SDD; el bug ODD declara delta o marcador |
| REQ-30.6 | Comprobación estructural del contexto técnico | Rechaza rutas `reference` e informa de las `maintained` |
| REQ-30.7 | Negativa en proyectos ISO | Remite al carril SDD sin escribir nada |
| REQ-30.8 | Multirepo | Opera sobre el repositorio de especificaciones |

## No cubre esta capacidad

- Que `axiom branch done` lea el documento ODD o fuerce la ejecución de la integración: ejecutar `integrate --odd` al cerrar es doctrina (capacidad `organic-agent-trigger-rules`, REQ-24.58).
- Juzgar si la delta describe correctamente el cambio o si la edición del contexto es correcta: lo cubre el modelo y la revisión.
- Identificadores funcionales (`RF-xxx`) como identidad de requisito y asignación de identificadores por Go.
- Push, pull request o merge desde Axiom.

---

## 1. Capacidad: `odd-living-spec-integration`

### Requirement: Sección `## Spec viva` del Documento ODD (REQ-30.1)

El documento ODD PUEDE contener una sección `## Spec viva` con la delta OpenSpec que el trabajo aplica a la spec viva. La sección DEBE contener un subapartado `### Capacidad: <dominio>` por cada capacidad afectada, y cada subapartado DEBE contener exactamente un bloque de código vallado con la delta en el formato OpenSpec (`## ADDED Requirements`, `## MODIFIED Requirements`, `## REMOVED Requirements` o `## RENAMED Requirements`, con `### Requirement:` y `#### Scenario:`). Los encabezados de la delta van dentro de la valla para que no cierren la sección del documento. El analizador DEBE rechazar con un motivo tipado que nombre el subapartado: un `### Capacidad:` sin bloque vallado, un bloque que no se pueda analizar como delta, dos subapartados de la misma capacidad, y una operación MODIFIED, REMOVED o RENAMED sobre una capacidad que no existe en `openspec/specs/`. Una operación ADDED sobre una capacidad inexistente la crea. Un documento sin la sección no integra nada. La sección NO DEBE alterar el listado ni el recuento de casillas del documento ODD.

#### Scenario: Una delta por capacidad se reconoce
- **DADO** un documento ODD con `## Spec viva` y dos subapartados `### Capacidad:`, cada uno con un bloque vallado con un `## ADDED Requirements`
- **CUANDO** se analiza el documento
- **ENTONCES** se reconocen las dos capacidades y sus deltas
- **Y** el listado de casillas del documento no cambia

#### Scenario: Un subapartado sin bloque vallado se rechaza
- **DADO** un `### Capacidad: facturacion` sin bloque de código vallado
- **CUANDO** se analiza el documento
- **ENTONCES** se rechaza con un motivo tipado que nombra `facturacion`

#### Scenario: Capacidad duplicada
- **DADO** dos subapartados `### Capacidad: facturacion`
- **CUANDO** se analiza el documento
- **ENTONCES** se rechaza nombrando la capacidad repetida

#### Scenario: MODIFIED sobre una capacidad inexistente
- **DADO** un subapartado cuya delta tiene un `## MODIFIED Requirements` sobre una capacidad que no existe en `openspec/specs/`
- **CUANDO** se analiza el documento
- **ENTONCES** se rechaza nombrando la capacidad y la sección

#### Scenario: ADDED crea una capacidad nueva
- **DADO** un subapartado cuya delta solo tiene `## ADDED Requirements` sobre una capacidad inexistente
- **CUANDO** se integra
- **ENTONCES** se crea `openspec/specs/<capacidad>/spec.md` con esos requisitos

#### Scenario: Un documento sin la sección no integra nada
- **DADO** un documento ODD sin `## Spec viva` ni `## Impacto en la spec` y que no es un bug
- **CUANDO** se ejecuta la integración
- **ENTONCES** termina con éxito sin modificar `openspec/specs/`

---

### Requirement: `axiom archive integrate --odd <feature>` Compone la Delta sobre la Spec Viva (REQ-30.2)

`axiom archive integrate --odd <feature>` DEBE leer `odd/tasks/<feature>.md` de la rama actual, validar sus secciones (REQ-30.1, REQ-30.5), comprobar las colisiones de identificadores (REQ-30.4) y la comprobación estructural del contexto (REQ-30.6) y componer cada delta sobre `openspec/specs/<capacidad>/spec.md` con la misma composición nativa que usa el archivado SDD (REQ-30.22 de `sdd-archive-lifecycle`). Todas las composiciones DEBEN validarse y componerse en memoria antes de escribir ninguna: o se aplican todas o ninguna. La orden DEBE dejar el índice `openspec/INDEX.md` coherente con las especificaciones integradas cuando el proyecto lo mantiene. La orden NO DEBE tener compuertas: ni paradas, ni preguntas, ni aprobación humana (REQ-27.17 de `organic-agent-trigger-rules`); NO DEBE crear artefactos SDD (`openspec/changes/<cambio>/`, `kickoff.yaml`, `gates.yaml`), NO DEBE escribir fuera de `openspec/specs/` e `openspec/INDEX.md` y NO DEBE modificar el documento ODD. Un documento inexistente, una sección inválida, una composición no aplicable o un rechazo de las comprobaciones DEBE finalizar con código distinto de cero y con un motivo tipado que nombre la sección o el requisito y la continuación, sin escribir nada.

Supuesto: la actualización de `openspec/INDEX.md` sigue el criterio de `living-documentation` (REQ-1.2): el diseño fija si la regeneración es siempre o solo cuando el índice ya existe.

#### Scenario: La delta se integra en la spec viva
- **DADO** un documento ODD con `## Spec viva` que añade un requisito `(REQ-31.1)` a la capacidad `facturacion`, sin colisión
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** `openspec/specs/facturacion/spec.md` contiene el requisito
- **Y** el código de salida es `0`
- **Y** no se crea ninguna carpeta bajo `openspec/changes/`

#### Scenario: Todo o nada entre capacidades
- **DADO** un documento ODD con dos capacidades, la segunda con un MODIFIED sobre un requisito que no existe
- **CUANDO** se ejecuta la integración
- **ENTONCES** finaliza con código distinto de cero nombrando la capacidad, la sección y el requisito
- **Y** ninguna especificación de `openspec/specs/` cambia, tampoco la de la primera capacidad

#### Scenario: Sin compuertas ni artefactos SDD
- **DADO** una integración válida
- **CUANDO** se ejecuta en una rama ODD
- **ENTONCES** no pide confirmación ni se detiene
- **Y** no se crean `kickoff.yaml`, `gates.yaml` ni un cambio SDD
- **Y** el documento ODD no se modifica

#### Scenario: Documento inexistente
- **DADO** una rama sin `odd/tasks/inexistente.md`
- **CUANDO** se ejecuta `axiom archive integrate --odd inexistente`
- **ENTONCES** finaliza con código distinto de cero con un motivo tipado que nombra la ruta esperada

---

### Requirement: Idempotencia de la Integración ODD (REQ-30.3)

La integración DEBE ser idempotente: si la spec canónica ya contiene cada requisito ADDED o MODIFIED de la delta con el mismo cuerpo, o ya no contiene cada requisito REMOVED, la orden DEBE terminar con éxito sin modificar ningún fichero e informar de que no había nada que integrar. Un requisito ya presente con un cuerpo distinto del de la delta DEBE rechazarse con un motivo tipado que nombre el requisito, sin escribir. Un requisito ya integrado NO DEBE contarse como colisión de identificadores (REQ-30.4). Tras un rebase sobre un cambio concurrente, si el nombre exacto de un requisito MODIFIED, REMOVED o RENAMED ya no casa con la spec canónica, la composición DEBE fallar cerrada y el autor corrige la delta.

#### Scenario: Repetir la integración no cambia nada
- **DADO** una integración ya aplicada
- **CUANDO** se repite `axiom archive integrate --odd <feature>`
- **ENTONCES** termina con código `0` informando de que no había nada que integrar
- **Y** `openspec/specs/` no cambia byte a byte

#### Scenario: Un requisito ya presente con otro cuerpo se rechaza
- **DADO** un requisito ADDED `(REQ-31.1)` ya integrado con un cuerpo que difiere del de la delta
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza con un motivo tipado que nombra el requisito
- **Y** no se escribe nada

#### Scenario: Un REMOVED ya aplicado es un no-op
- **DADO** una delta con un REMOVED de un requisito que ya no está en la spec canónica
- **CUANDO** se ejecuta la integración
- **ENTONCES** termina con código `0` sin modificar ningún fichero

#### Scenario: Tras un rebase el nombre ya no casa
- **DADO** una delta con un MODIFIED cuyo requisito fue renombrado por un cambio concurrente ya fusionado
- **CUANDO** se ejecuta la integración
- **ENTONCES** falla cerrada nombrando el requisito que no se aplica
- **Y** no se escribe nada

---

### Requirement: Comprobación de Colisión de Identificadores `REQ-n.m` (REQ-30.4)

Todo requisito ADDED de una delta ODD DEBE llevar en su título un identificador `(REQ-n.m)`, elegido por el autor (el siguiente número libre de la serie). La integración DEBE rechazar sin escribir un requisito ADDED cuyo identificador ya existe en `openspec/specs/` o en una delta activa de `openspec/changes/<cambio>/specs/` (la carpeta `archive/` queda excluida), así como dos requisitos ADDED con el mismo identificador dentro del propio documento. El motivo tipado DEBE nombrar el identificador, dónde colisiona (la capacidad o el cambio) y el siguiente número libre de esa serie `n`. Un requisito ADDED sin identificador `REQ-n.m` en el título DEBE rechazarse nombrando el requisito. Un requisito que la spec canónica ya contiene con el mismo cuerpo no es una colisión (REQ-30.3).

Supuestos (propuesta §13, P5): el alcance de la comprobación es `openspec/specs/` y las deltas de los cambios activos, sin `archive/`; la obligación de que todo ADDED lleve identificador es una decisión de esta spec porque el indexador exige `REQ-` (`living-documentation`, REQ-1.1).

#### Scenario: Colisión con la spec viva
- **DADO** una delta ODD con un ADDED `(REQ-31.1)` y un `REQ-31.1` ya existente en `openspec/specs/facturacion/spec.md`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando `REQ-31.1`, la capacidad `facturacion` y el siguiente número libre `REQ-31.2`
- **Y** no se escribe nada

#### Scenario: Colisión con la delta de un cambio activo
- **DADO** un ADDED `(REQ-31.2)` y un cambio SDD activo cuya delta ya añade `REQ-31.2`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando `REQ-31.2`, el cambio activo y el siguiente número libre

#### Scenario: Los cambios archivados no cuentan
- **DADO** un `REQ-31.3` presente solo en `openspec/changes/archive/<cambio>/specs/`
- **CUANDO** se integra un ADDED `(REQ-31.3)`
- **ENTONCES** no hay colisión por ese motivo

#### Scenario: Duplicado dentro del documento
- **DADO** dos subapartados con un ADDED `(REQ-31.4)` cada uno
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando `REQ-31.4` y ambas capacidades

#### Scenario: ADDED sin identificador
- **DADO** un ADDED cuyo título no contiene `(REQ-n.m)`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando el requisito

#### Scenario: Una integración ya aplicada no colisiona consigo misma
- **DADO** un ADDED `(REQ-31.1)` ya integrado con el mismo cuerpo
- **CUANDO** se repite la integración
- **ENTONCES** no se reporta colisión y termina con código `0`

---

### Requirement: Marcador `Sin impacto en la spec` y Obligación de los Bugs ODD (REQ-30.5)

El documento ODD PUEDE contener una sección de nivel 2 `## Impacto en la spec` y un bug ODD (cabecera con la línea `Tipo: bug`) DEBE declarar su impacto. El marcador de «sin impacto» es el mismo contrato que el carril SDD (REQ-29.15 de `axiom-sdd-cli-integration`, segunda enmienda de `carril-sdd-versatil`): una sección de nivel 2 `## Impacto en la spec` que contiene la línea literal `Sin impacto en la spec` y una línea `Motivo: <una línea>` con texto no vacío. La línea del marcador DEBE compararse de forma exacta y sensible a mayúsculas, ignorando únicamente los espacios iniciales y finales: `Sin impacto en la spec.` (con punto final) o `sin impacto en la spec` NO son el marcador. Un `Motivo:` ausente o vacío invalida el marcador. Reglas de `axiom archive integrate --odd`:

- Un bug con una sección `## Spec viva` con contenido (al menos una capacidad) se evalúa como un cambio con delta y se integra (REQ-30.2); no necesita marcador.
- Un bug sin `## Spec viva` con contenido DEBE llevar el marcador válido; con él, la orden termina con éxito sin modificar la spec viva e informa del motivo. Sin marcador, o con un marcador inválido, DEBE rechazarse con un motivo tipado que nombre la sección y la regla incumplida, sin escribir nada.
- Marcador válido junto con una `## Spec viva` con contenido: el marcador se ignora y el documento se evalúa como un cambio con delta (la misma regla que el carril SDD).
- En un documento que no es un bug, la sección y el marcador son opcionales y NO DEBEN producir rechazo.

Diferencia explícita entre carriles: el bug ODD se señala con `Tipo: bug` en la cabecera y NO DEBE exigir la sección `## Informe de bug` del carril SDD; el contrato del marcador es el mismo en ambos.

Supuestos (propuesta §13, P3): un bug ODD se marca con la línea `Tipo: bug`. La forma «Ver `## Spec viva`» de la versión previa de esta spec desaparece: con delta no hace falta declaración.

#### Scenario: Un bug sin delta ni marcador se rechaza
- **DADO** un documento con `Tipo: bug`, sin `## Spec viva` con contenido y sin `## Impacto en la spec`
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** se rechaza con un motivo tipado que exige la delta o el marcador
- **Y** no se escribe nada

#### Scenario: El marcador con motivo pasa sin cambiar la spec
- **DADO** un bug con `## Impacto en la spec` que contiene la línea `Sin impacto en la spec` y la línea `Motivo: el fallo era un error tipográfico en un mensaje y ningún requisito lo cubre`
- **CUANDO** se ejecuta la integración
- **ENTONCES** termina con código `0` informando del motivo
- **Y** `openspec/specs/` no cambia

#### Scenario: Motivo vacío o ausente invalida el marcador
- **DADO** un bug con la línea `Sin impacto en la spec` y con `Motivo:` sin texto, o sin línea `Motivo:`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando la sección y la falta de motivo

#### Scenario: La comparación es exacta y sensible a mayúsculas
- **DADO** un bug con la línea `Sin impacto en la spec.` (con punto final) y otro con `sin impacto en la spec`, ambos con `Motivo:` no vacío
- **CUANDO** se ejecuta la integración sobre cada uno
- **ENTONCES** ninguno se reconoce como marcador y ambos se rechazan

#### Scenario: Los espacios iniciales y finales se ignoran
- **DADO** un bug con la línea `  Sin impacto en la spec  ` y `Motivo:` no vacío
- **CUANDO** se ejecuta la integración
- **ENTONCES** el marcador es válido

#### Scenario: Un bug con delta se integra sin marcador
- **DADO** un bug con una `## Spec viva` con una capacidad y sin `## Impacto en la spec`
- **CUANDO** se ejecuta la integración
- **ENTONCES** la delta se integra como en REQ-30.2

#### Scenario: El marcador junto con una delta con contenido se ignora
- **DADO** un bug con el marcador válido y una `## Spec viva` con contenido
- **CUANDO** se ejecuta la integración
- **ENTONCES** el marcador se ignora y se evalúa como un cambio con delta
- **Y** la delta se integra

#### Scenario: Un bug ODD no exige `## Informe de bug`
- **DADO** un bug ODD con `Tipo: bug`, sin la sección `## Informe de bug`, con delta o con marcador válido
- **CUANDO** se ejecuta la integración
- **ENTONCES** no se rechaza por la falta de esa sección

#### Scenario: Una feature que no es bug no exige la sección
- **DADO** un documento sin `Tipo: bug` y sin `## Impacto en la spec`
- **CUANDO** se ejecuta la integración
- **ENTONCES** no se rechaza por la falta de la sección

---

### Requirement: Comprobación Estructural del Contexto Técnico en la Integración ODD (REQ-30.6)

Cuando `axiom.yaml` declara `technical_context`, `axiom archive integrate --odd` DEBE calcular los ficheros que la rama cambió frente a su base y: (a) rechazar sin escribir que la rama modifique, cree o elimine ficheros bajo `technical_context.reference.paths` (REQ-24.30 de `project-technical-context`), nombrando cada fichero; y (b) informar, sin bloquear, de si la rama tocó alguna ruta de `technical_context.maintained.paths`, nombrando las rutas tocadas o indicando que ninguna lo fue. La orden NO DEBE exigir ningún marcador «sin impacto en el contexto» ni juzgar el contenido de la edición. Sin `technical_context` declarado la comprobación NO DEBE producir ni error ni informe. Si la base de la rama no puede determinarse, la orden DEBE rechazar con un motivo tipado que lo indique, porque no puede garantizar la regla de referencia.

Supuestos (propuesta §13, P4): no hay marcador de impacto en el contexto en ODD; el rechazo cuando la base no es determinable (fallo cerrado) es una decisión de esta spec.

#### Scenario: Un cambio en una ruta de referencia se rechaza
- **DADO** un `technical_context.reference.paths` con `docs/referencia.md` y una rama que modifica ese fichero
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando `docs/referencia.md` y el modo `reference`
- **Y** no se escribe nada

#### Scenario: Rutas mantenidas tocadas se informan sin bloquear
- **DADO** un `technical_context.maintained.paths` con `context/` y una rama que modifica `context/ARQUITECTURA.md`
- **CUANDO** se ejecuta la integración
- **ENTONCES** la salida informa de que se tocó `context/ARQUITECTURA.md`
- **Y** la integración continúa

#### Scenario: Rutas mantenidas sin tocar se informan
- **DADO** el mismo contexto y una rama que no toca `context/`
- **CUANDO** se ejecuta la integración
- **ENTONCES** la salida informa de que no se tocó ninguna ruta mantenida
- **Y** no se bloquea ni se exige ningún marcador

#### Scenario: Sin contexto técnico declarado no hay comprobación
- **DADO** un `axiom.yaml` sin `technical_context`
- **CUANDO** se ejecuta la integración
- **ENTONCES** la salida no contiene informe ni error del contexto

#### Scenario: Base de la rama no determinable
- **DADO** un `technical_context` declarado y una rama cuya base no puede determinarse
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza con un motivo tipado que nombra la causa
- **Y** no se escribe nada

---

### Requirement: Negativa de la Integración ODD en un Proyecto ISO (REQ-30.7)

Cuando `axiom sdd status` proyecta que el proyecto declara ISO (REQ-29.2 de `sdd-iso-evidence`), `axiom archive integrate --odd` DEBE negarse con un motivo tipado, sin escribir nada, que remita al carril SDD: los bugs por el perfil de bug y los incrementos como cambio SDD, porque ODD no deja `Cubre:` ni matriz de verify. En un proyecto no ISO la orden NO DEBE verse afectada por esta regla.

Supuesto (propuesta §13, P2): la negativa es obligatoria en proyectos ISO. Este requisito depende de la declaración ISO que introduce `carril-sdd-versatil`; hasta que esa declaración exista en `main` no hay proyecto ISO y el requisito no tiene efecto observable.

#### Scenario: Un proyecto ISO rechaza la integración ODD
- **DADO** un proyecto cuyo status proyecta que declara ISO y un documento ODD válido
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** se rechaza con un motivo tipado que remite al carril SDD (bug con el perfil de bug, incremento como cambio)
- **Y** `openspec/specs/` no cambia

#### Scenario: Un proyecto no ISO no se ve afectado
- **DADO** un proyecto que no declara ISO y un documento ODD válido
- **CUANDO** se ejecuta la integración
- **ENTONCES** se integra con normalidad (REQ-30.2)

---

### Requirement: En Multirepo la Integración ODD Opera sobre el Repositorio de Especificaciones (REQ-30.8)

En una topología `multirepo`, `axiom archive integrate --odd` DEBE resolver `workspaceRoot` con el primitivo de la capacidad `planning-home-resolution` y operar sobre él: leer el documento ODD, que en multirepo vive en el repositorio de especificaciones, y escribir `openspec/specs/` de ese repositorio. NO DEBE modificar ningún repositorio de código. En `monorepo-embedded` opera sobre el repositorio actual.

Supuesto (propuesta §13, P6): `workspaceRoot` es la raíz de operación.

#### Scenario: Multirepo escribe en el repositorio de especificaciones
- **DADO** un `multirepo` con `specs_repository: repo-specs`, un documento ODD en ese repositorio y una ejecución desde la carpeta maestra
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** se integra en `repo-specs/openspec/specs/`
- **Y** ningún repositorio de código cambia

#### Scenario: Monorepo embebido opera en el repositorio actual
- **DADO** un `monorepo-embedded` con el documento ODD en la rama actual
- **CUANDO** se ejecuta la integración
- **ENTONCES** se escribe `openspec/specs/` del mismo repositorio
