# ODD: `axiom trace` encuentra por título los incrementos de los requisitos sin ID antiguo (trace-titulo-sin-id)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/trace-titulo-sin-id.md`. Espejo Engram: `odd/trace-titulo-sin-id/tasks`.
> Rama: `fix/trace-titulo-sin-id` · worktree `C:\repos\axiom-wt\trace-titulo-sin-id\axiom`.

Tipo: bug

## Objetivo

Que `axiom trace <ID nuevo>` de un requisito que no tenía ID antiguo encuentre los incrementos archivados y los documentos ODD que lo tocaron por su título.

## Problema y porqué

La migración C-03-e (`migrar-ids-rf`) dio un ID nuevo a 48 requisitos que no tenían ID. El mapa `openspec/id-map.yaml` guarda para ellos el título en lugar del ID antiguo (RF-TRC-16), pero la traza solo casa IDs. Por eso `axiom trace RF-ENG-01` no ve ninguno de los incrementos anteriores a la migración, que citaban el requisito solo por su título. Afecta a 36 de los 48: tienen al menos un cambio archivado o un documento ODD cuya delta los nombra por título. Pendiente informativo del archive de `migrar-ids-rf` (`archive-report.md`, hueco 6 de `coverage-review.design.md`).

## Decisiones

- **Fuente del título:** la entrada `title` del mapa (capacidad, título, ID nuevo). No se lee la spec viva: el mapa es el registro inmutable de la migración y ya lo carga la traza.
- **Acotado por capacidad:** el título solo casa en una delta de la misma capacidad que la entrada (carpeta `specs/<capacidad>/` o `### Capacidad:` del documento ODD), igual que los IDs antiguos (RF-TRC-04, regla 3). Una línea `Cubre:` cita IDs, nunca títulos, así que no participa.
- **Comparación:** el nombre del requisito de la delta, sin espacios al principio ni al final, idéntico al título del mapa. Sin normalizar mayúsculas: el mapa copia el título original byte a byte. En RENAMED casan el nombre origen y el destino.
- **Sin cadena de renombrados:** un título anterior a un RENAMED que no figure en el mapa no se sigue. Es un caso límite y seguirlo exigiría ordenar los cambios en el tiempo.
- **ID anotado:** el incremento lleva el ID nuevo en `ids`, porque el título no tiene token de ID.

## Alcance autorizado

`internal/legacyid` (índice de títulos), `internal/reqtrace` (coincidencia por título), sus pruebas y la spec viva `requirement-trace` (RF-TRC-04).

## Modo TDD

Activo por elección del usuario para los arreglos de esta sesión (RED observado antes del arreglo). Ejecutor: `go test ./internal/reqtrace/ ./internal/legacyid/ -run <prueba>`. La suite entera la pasa la CI.

## Entrega

`single-pr`: una PR con un commit por tarea. Etiqueta `type:bug`.

## Tareas

- [x] T1 · `legacyid.Map.Title` y coincidencia por título en `classifyDelta`, con pruebas (SDD y ODD, misma capacidad, otra capacidad, RENAMED, título parecido) y la spec viva integrada con `axiom archive integrate --odd trace-titulo-sin-id`. Ruta: inline (unidad pequeña, menos de 100 líneas). RED observado: `TestRequirementTraceLegacyTitleEntryMatchesByTitle` falló con `incrementos = map[e:created]` antes del arreglo. Evidencia: `go test ./internal/reqtrace/ ./internal/legacyid/` ok; `go test ./internal/cli/ -run Trace` ok; `go test ./internal/openspecdoc/ ./internal/reqid/ ./internal/livespec/` ok; `TestTraceRealMapFindsTitleEntryIncrements` encuentra por título los archivados de RF-ENG-01 en el corpus real. Integración `composed` de `requirement-trace` (más `INDEX.md`). Corrección de CI: `TestRepositoryLegacyIDResidues` exigía declarar `REQ-1.1` y `REQ-24.3` del bloque `openspec` de este documento; residuo `md-code` añadido a `internal/idmigration/residues.yaml` (`go test ./internal/idmigration/` ok).

## Criterio de terminado

- `axiom trace` por el ID nuevo de un requisito con entrada `title` devuelve los incrementos cuya delta de su capacidad lo nombra por título, con su rol.
- Una delta de otra capacidad con el mismo título no cuenta.
- Las pruebas de los paquetes tocados pasan y la CI está en verde.

## Spec viva

### Capacidad: requirement-trace

```openspec
## MODIFIED Requirements

### Requirement: Punto de Extensión de Alias (RF-TRC-04)

La resolución de equivalentes MUST pasar por un punto de extensión de alias sustituible, de modo que el mapa de identificadores heredados (RF-TRC-16) añada equivalentes sin modificar la consulta. La implementación por defecto MUST devolver únicamente el propio ID, su pareja provisional/definitivo y los `Previous` de las trazas. El mapa de identificadores heredados MUST estar cableado como resolutor de alias de `axiom trace` cuando existe, con estas reglas:

1. Un ID antiguo MUST resolverse a todos sus IDs nuevos: un ID antiguo repetido en varias capacidades (`REQ-1.1`) devuelve el ID nuevo de cada una de ellas.
2. Un ID nuevo MUST resolverse a su ID antiguo (cuando lo tiene), para encontrar los incrementos archivados que lo citan.
3. La coincidencia de un ID antiguo en los incrementos archivados MUST acotarse por la capacidad: un ID antiguo que aparece en el título de una delta archivada de la capacidad `C` (carpeta `specs/C/`), o en una línea `Cubre:` o una entrada de `ids.yaml` cuyo contexto nombra `C`, equivale solo al ID nuevo de (`C`, ID antiguo). MUST NOT contar como incremento del ID nuevo de otra capacidad que usaba el mismo ID antiguo.
4. Consultar por un ID nuevo MUST traer los incrementos archivados que tocaron su ID antiguo en su capacidad, con los roles de RF-TRC-05, además de los que citan el ID nuevo.
5. Consultar por el ID nuevo de una entrada del mapa sin ID antiguo (la que lleva el título) MUST traer también los incrementos, SDD archivados u ODD, cuya delta de esa misma capacidad nombra un requisito con ese título exacto (sin espacios al principio ni al final), con los roles de RF-TRC-05; en RENAMED cuentan el nombre origen y el destino. Una delta de otra capacidad con el mismo título MUST NOT contar, y las líneas `Cubre:` no casan por título.

Un alias que no sea válido bajo el patrón del proyecto MUST aceptarse cuando proviene del mapa de identificadores heredados.

#### Scenario: Un alias inyectado se incorpora a la consulta

- **DADO** un resolutor de alias que declara `«ID-b»` equivalente a `«ID»`
- **CUANDO** se consulta `«ID»`
- **ENTONCES** los IDs resueltos incluyen `«ID-b»`
- **Y** los incrementos que tocan `«ID-b»` forman parte del resultado

#### Scenario: Sin resolutor adicional solo cuentan las trazas

- **DADO** la implementación por defecto y un ID sin entrada de traza
- **CUANDO** se consulta
- **ENTONCES** los IDs resueltos son solo el ID consultado

#### Scenario: Un ID antiguo ambiguo resuelve a todos sus IDs nuevos

- **DADO** el mapa con `REQ-1.1` en siete capacidades
- **CUANDO** se consulta `axiom trace REQ-1.1`
- **ENTONCES** los IDs resueltos incluyen el ID nuevo de cada una de las siete capacidades y el propio `REQ-1.1`

#### Scenario: Un ID antiguo no ambiguo resuelve a su ID nuevo

- **DADO** el mapa con `REQ-24.3` en una sola capacidad
- **CUANDO** se consulta `axiom trace REQ-24.3`
- **ENTONCES** los IDs resueltos incluyen el ID nuevo de esa capacidad y `REQ-24.3`

#### Scenario: La coincidencia en archivados se acota por capacidad

- **DADO** una delta archivada de la capacidad `autoskills` que modifica `REQ-1.1` y otra de `workspace-topology` que modifica el mismo `REQ-1.1`
- **CUANDO** se consulta el ID nuevo de `autoskills` para ese requisito
- **ENTONCES** el resultado incluye el incremento de `autoskills`
- **Y** no incluye el de `workspace-topology`

#### Scenario: Consultar por el ID nuevo trae los incrementos del ID antiguo

- **DADO** un cambio archivado cuya delta de la capacidad `C` modifica `«ID-antiguo»` y el mapa que lo asocia a `«ID-nuevo»` en `C`
- **CUANDO** se consulta `«ID-nuevo»`
- **ENTONCES** el resultado incluye ese cambio archivado con el rol `modified`

#### Scenario: Un requisito sin ID antiguo se encuentra por su título en su capacidad

- **DADO** el mapa con la entrada (`C`, título `«Título»`, `«ID-nuevo»`) sin ID antiguo, un cambio archivado cuya delta de `C` modifica `### Requirement: «Título»` y otro cuya delta de otra capacidad modifica un requisito con el mismo título
- **CUANDO** se consulta `«ID-nuevo»`
- **ENTONCES** el resultado incluye el primer cambio con el rol `modified`
- **Y** no incluye el segundo
```
