# Especificación de Requerimientos: Adopción de OpenSpec como Spec Viva Única (REQ-30.9 a REQ-30.14)

> **Cambio:** `spec-viva-openspec` · **Tipo:** capacidad nueva (especificación completa, no delta)
> **Idioma:** Español (castellano peninsular)

## Propósito

Definir cómo un proyecto que llega con documentación de requisitos previa, en cualquier formato, la migra a `openspec/specs/` mediante un cambio SDD de adopción: un inventario determinista de los orígenes (`axiom archive adopt scan`), un formato del cambio de adopción y una verificación determinista de ese cambio (`axiom archive adopt check`), con la aprobación humana de la compuerta `spec`.

**Resultado esperado:** toda unidad del origen queda migrada a un requisito OpenSpec con su procedencia (`Origen:`) o descartada con un motivo explícito, con identificadores únicos y al menos un escenario por requisito, y un humano ha aprobado la migración.

## Términos

| Término | Definición |
|---|---|
| Origen | Fichero o conjunto de ficheros de requisitos previos al proyecto, fuera de `openspec/specs/`. |
| Unidad | Parte direccionable de un origen: un fichero o un encabezado, identificada por ruta y ancla. |
| Inventario | Lista determinista de orígenes y unidades que `adopt scan` produce; es la referencia de cobertura. |
| Cambio de adopción | Cambio SDD cuya delta solo contiene `## ADDED Requirements` que migran unidades del inventario. |
| `Origen:` | Línea del cuerpo de un requisito migrado con la ruta, el ancla y el identificador original de su unidad. |
| `Descartado:` | Lista de unidades del inventario que no se migran, cada una con su motivo. |

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| REQ-30.9 | `axiom archive adopt scan` | Inventario determinista; sin `--change` no escribe nada |
| REQ-30.10 | Formato del cambio de adopción | Solo ADDED, `Origen:` por requisito, `Descartado:` con motivo |
| REQ-30.11 | `axiom archive adopt check --change <c>` | Verificaciones deterministas con todos los fallos reportados |
| REQ-30.12 | Series de identificadores | Una serie nueva por proyecto migrado; `RF-xxx` solo en `Origen:` |
| REQ-30.13 | Aprobación humana y archivado | La compuerta `spec` aprueba; `Origen:` se conserva al archivar |
| REQ-30.14 | Multirepo | Opera sobre el repositorio de especificaciones |

## No cubre esta capacidad

- Que Go juzgue la fidelidad semántica de la migración: cubre la compuerta `spec` humana.
- Importadores por formato (Word, hojas de cálculo) y cualquier destino distinto de OpenSpec.
- Identificadores funcionales (`RF-xxx`) como identidad de requisito y migrar los `REQ-x.y` propios de Axiom.
- Que el dashboard ejecute modelos: solo muestra el informe y la instrucción (`multi-project-hub`).
- Que `axiom init` escriba `technical_context:` por defecto: solo se propone en el informe de `adopt scan`.

---

## 1. Capacidad: `openspec-adoption`

### Requirement: `axiom archive adopt scan` Inventaría los Orígenes (REQ-30.9)

`axiom archive adopt scan [--path <dir>] [--change <cambio>] [--json]` DEBE producir un inventario de solo lectura de los orígenes candidatos del proyecto (ficheros de requisitos fuera de `openspec/specs/` y de `openspec/changes/`) y de sus unidades: para cada origen, sus ficheros, sus encabezados, los identificadores ajenos reconocidos por patrón (por ejemplo `RF-120`) y los candidatos a `technical_context` (documentación mantenida y de referencia) propuestos para declarar en `axiom.yaml`. El inventario DEBE ser determinista: el mismo árbol produce el mismo inventario con el mismo orden y las mismas anclas. Sin `--change`, la orden NO DEBE escribir ningún fichero. Con `--change <cambio>`, DEBE escribir únicamente el inventario en la carpeta del cambio (`adoption-inventory.yaml`, formato `axiom.openspec-adoption-inventory/v1`) y nada más. Como escritor Go de la carpeta del cambio, DEBE consultar la guarda A de REQ-25.19 (`change-branch-lifecycle`) en el contexto ordinario antes de escribir y, ante `wrong_branch`, DEBE negarse sin escribir nada y devolver la continuación de `axiom branch ensure`. Sin ningún origen candidato, la orden DEBE informarlo y finalizar con código `0`. La orden NO DEBE requerir que exista `openspec/`.

Supuesto: el inventario con `--change` es la referencia de cobertura de REQ-30.11; sin esa opción, el detector del hub lo reutiliza sin escribir (REQ-30.26 de `multi-project-hub`).

#### Scenario: El inventario lista orígenes y unidades sin escribir
- **DADO** un proyecto con `docs/requisitos.md` con dos encabezados que contienen `RF-001` y `RF-002`
- **CUANDO** se ejecuta `axiom archive adopt scan`
- **ENTONCES** el inventario lista el origen, sus dos unidades con su ruta y ancla, y los identificadores `RF-001` y `RF-002`
- **Y** no se crea ni se modifica ningún fichero

#### Scenario: El inventario es determinista
- **DADO** el mismo árbol de ficheros
- **CUANDO** se ejecuta `adopt scan --json` dos veces
- **ENTONCES** las dos salidas son idénticas

#### Scenario: Con `--change` solo se escribe el inventario
- **DADO** un cambio `adopcion-requisitos`
- **CUANDO** se ejecuta `axiom archive adopt scan --change adopcion-requisitos`
- **ENTONCES** la carpeta del cambio contiene el inventario
- **Y** ningún otro fichero del proyecto cambia

#### Scenario: `adopt scan --change` se niega fuera de rama
- **DADO** un cambio `adopcion-requisitos` con instantánea y el repositorio desplegado en la rama de integración
- **CUANDO** se ejecuta `axiom archive adopt scan --change adopcion-requisitos`
- **ENTONCES** se niega con `wrong_branch` y la continuación de `axiom branch ensure`
- **Y** no se crea el inventario ni ningún otro fichero de la carpeta del cambio

#### Scenario: Sin orígenes candidatos
- **DADO** un proyecto sin documentación de requisitos fuera de `openspec/`
- **CUANDO** se ejecuta `adopt scan`
- **ENTONCES** informa de que no hay orígenes candidatos y el código de salida es `0`

#### Scenario: Candidatos a `technical_context`
- **DADO** un proyecto con documentación técnica y de requisitos sin bloque `technical_context:`
- **CUANDO** se ejecuta `adopt scan`
- **ENTONCES** el informe propone candidatos a `maintained` y a `reference` sin modificar `axiom.yaml`

---

### Requirement: Formato del Cambio de Adopción (REQ-30.10)

Un cambio de adopción es un cambio SDD ordinario (kickoff, compuertas y archivado normales) cuya delta de spec DEBE contener únicamente `## ADDED Requirements`. Cada requisito DEBE llevar en su título un identificador `(REQ-n.m)` y en su cuerpo al menos una línea `Origen: <ruta>#<ancla> (<ID original>)`, donde `<ID original>` es el identificador de la unidad en el origen cuando existe. Las unidades del inventario que no se migran DEBEN figurar en una lista `Descartado:` del cambio, cada una con su motivo (`<ruta>#<ancla>: <motivo>`, no vacío); la ubicación exacta de la lista dentro de la carpeta del cambio la fija el diseño. La composición al archivar conserva el texto `Origen:` del cuerpo sin tratamiento especial.

#### Scenario: Requisito migrado con su procedencia
- **DADO** un requisito `(REQ-40.1)` con `Origen: docs/requisitos.md#rf-001 (RF-001)` y un escenario
- **CUANDO** se analiza el cambio de adopción
- **ENTONCES** el requisito se reconoce con su identificador, su procedencia y su identificador original

#### Scenario: Una unidad descartada con motivo
- **DADO** una lista `Descartado:` con `docs/requisitos.md#glosario: no es un requisito`
- **CUANDO** se analiza el cambio
- **ENTONCES** la unidad se reconoce como descartada con su motivo

#### Scenario: Un requisito puede citar varios orígenes
- **DADO** un requisito con dos líneas `Origen:` que apuntan a dos unidades
- **CUANDO** se analiza el cambio
- **ENTONCES** ambas unidades cuentan como cubiertas por ese requisito

---

### Requirement: `axiom archive adopt check --change <cambio>` Verifica el Cambio de Adopción (REQ-30.11)

`axiom archive adopt check --change <cambio> [--path <dir>] [--json]` DEBE verificar de forma determinista, sin modelo y sin escribir ningún fichero: (a) formato: la delta solo contiene `## ADDED Requirements` y cada requisito tiene `(REQ-n.m)` en su título; (b) unicidad: ningún `REQ-n.m` se repite dentro del cambio ni coincide con uno de `openspec/specs/`; (c) escenarios: cada requisito tiene al menos un `#### Scenario:`; (d) procedencia: cada requisito tiene al menos una línea `Origen:` y cada `Origen:` resuelve a un fichero y a un ancla existentes; (e) cobertura: cada unidad del inventario figura en algún `Origen:` o en `Descartado:` con motivo no vacío. La orden DEBE reportar todos los fallos, no solo el primero, cada uno con un motivo tipado que nombre el requisito o la unidad y la regla incumplida, y finalizar con código distinto de cero si hay alguno. Sin inventario en la carpeta del cambio, DEBE rechazar con un motivo tipado que remita a `axiom archive adopt scan --change <cambio>`.

#### Scenario: Un cambio válido pasa
- **DADO** un cambio de adopción con solo ADDED, identificadores únicos, escenarios, `Origen:` resolubles y todas las unidades cubiertas o descartadas con motivo
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** finaliza con código `0`

#### Scenario: Un MODIFIED se rechaza
- **DADO** una delta de adopción con un `## MODIFIED Requirements`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando la sección y la regla de formato

#### Scenario: Identificador duplicado
- **DADO** dos requisitos `(REQ-40.1)` en el cambio, o uno que coincide con un `REQ-40.1` de `openspec/specs/`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando el identificador y dónde se repite

#### Scenario: Requisito sin escenario
- **DADO** un requisito sin ningún `#### Scenario:`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando el requisito

#### Scenario: `Origen:` que no resuelve
- **DADO** un `Origen:` con una ruta inexistente y otro con un ancla que no existe en el fichero
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando cada `Origen:`, la ruta o el ancla que no resuelve

#### Scenario: Unidad sin cubrir ni descartar
- **DADO** una unidad del inventario ausente de todo `Origen:` y de `Descartado:`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando la unidad

#### Scenario: Descartado sin motivo
- **DADO** una unidad en `Descartado:` sin motivo
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando la unidad y la falta de motivo

#### Scenario: Se reportan todos los fallos
- **DADO** un cambio con un requisito sin escenario y una unidad sin cubrir
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** el informe contiene ambos fallos

#### Scenario: Sin inventario
- **DADO** un cambio de adopción sin inventario en su carpeta
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** se rechaza remitiendo a `axiom archive adopt scan --change <cambio>`

---

### Requirement: Series de Identificadores de la Migración (REQ-30.12)

Todos los requisitos de un cambio de adopción DEBEN usar una única serie `REQ-n.*`, nueva para el proyecto migrado, cuyo `n` no se use en `openspec/specs/`; `adopt check` DEBE rechazar un cambio que mezcle series o que use una serie ya existente en `openspec/specs/`. El identificador original del origen (por ejemplo `RF-120`) NO es la identidad del requisito migrado: se conserva solo en `Origen:`. Los identificadores `REQ-x.y` propios de Axiom y los identificadores funcionales como identidad quedan fuera de esta capacidad.

Supuesto: el nombre y el número de la serie los elige el autor, como en el resto de OpenSpec; la regla de serie única por cambio es una decisión de esta spec que materializa «una serie nueva por proyecto migrado».

#### Scenario: Una sola serie nueva
- **DADO** un cambio con requisitos `REQ-40.1` a `REQ-40.9` y una `openspec/specs/` sin serie `40`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** no se reporta ningún fallo de serie

#### Scenario: Mezcla de series
- **DADO** un cambio con `REQ-40.1` y `REQ-41.1`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando las dos series

#### Scenario: Serie ya existente
- **DADO** un cambio con `REQ-28.40` y una serie `28` ya presente en `openspec/specs/`
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** falla nombrando la serie y la capacidad donde existe

#### Scenario: El identificador funcional no es la identidad
- **DADO** un requisito `(REQ-40.1)` con `Origen: docs/requisitos.md#rf-120 (RF-120)`
- **CUANDO** se archiva el cambio
- **ENTONCES** la spec viva identifica el requisito por `REQ-40.1`
- **Y** `RF-120` consta solo en la línea `Origen:`

---

### Requirement: Aprobación Humana de la Migración y Conservación de `Origen:` al Archivar (REQ-30.13)

La compuerta `spec` del proyecto que migra DEBE ser la aprobación humana de la fidelidad de la migración: `adopt check` verifica la forma y la cobertura, pero NO DEBE registrar ni sustituir ninguna decisión de compuerta ni juzgar la fidelidad semántica. El cambio de adopción DEBE archivarse con la composición `axiom sdd archive-compose --canonical --delta` y la spec viva resultante DEBE conservar las líneas `Origen:` de cada requisito.

Supuesto: la spec no hace que `adopt check` condicione el archivado ni la proyección de `axiom sdd status`; la verificación es una orden explícita que la doctrina de la skill de adopción manda ejecutar antes de la compuerta.

#### Scenario: `adopt check` no registra decisiones de compuerta
- **DADO** un cambio de adopción válido
- **CUANDO** se ejecuta `adopt check`
- **ENTONCES** el registro de compuertas del cambio no cambia

#### Scenario: El archivado conserva `Origen:`
- **DADO** un cambio de adopción válido con la compuerta `spec` aprobada
- **CUANDO** se archiva con `archive-compose --canonical --delta`
- **ENTONCES** cada requisito de `openspec/specs/<capacidad>/spec.md` conserva su línea `Origen:` byte a byte

---

### Requirement: En Multirepo la Adopción Opera sobre el Repositorio de Especificaciones (REQ-30.14)

En una topología `multirepo`, `axiom archive adopt scan` y `axiom archive adopt check` DEBEN resolver `workspaceRoot` con el primitivo de la capacidad `planning-home-resolution` y operar sobre el repositorio de especificaciones: el inventario cubre los orígenes de ese repositorio, el cambio de adopción vive en su `openspec/changes/` y `openspec/specs/` es el de ese repositorio. En `monorepo-embedded` operan sobre el repositorio actual.

Supuesto (propuesta §13, P6): `workspaceRoot` es la raíz de operación; los orígenes que vivan en repositorios de código hermanos quedan fuera del inventario.

#### Scenario: Multirepo inventaría el repositorio de especificaciones
- **DADO** un `multirepo` con `specs_repository: repo-specs` y requisitos previos en `repo-specs/docs/`
- **CUANDO** se ejecuta `adopt scan --path <maestra>`
- **ENTONCES** el inventario contiene los orígenes de `repo-specs/docs/`
- **Y** no contiene ficheros de los repositorios de código

#### Scenario: Monorepo embebido
- **DADO** un `monorepo-embedded`
- **CUANDO** se ejecuta `adopt scan`
- **ENTONCES** el inventario cubre el repositorio actual
