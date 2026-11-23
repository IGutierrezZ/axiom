# Delta para odd-living-spec-integration

> Parte del estado vigente de `openspec/specs/odd-living-spec-integration/spec.md`. REQ-30.4 deja de fijar la forma literal `(REQ-n.m)` y pide un identificador válido según el patrón declarado del proyecto (REQ-44.1 a REQ-44.4 de `requirement-identifiers`); con el patrón por defecto el comportamiento no cambia. REQ-30.4 cambia de título: se expresa con RENAMED seguido de MODIFIED bajo el título nuevo, para que el archivado funcione con el compositor que empareja por título exacto y con el que empareja por identificador (REQ-30.22 de `sdd-archive-lifecycle`).
>
> La tabla «Requerimientos de un vistazo» de esta capacidad (que cita `REQ-n.m` en REQ-30.4) no es un bloque de requisito y no se edita con una delta; describe el patrón por defecto.
>
> REQ-44.8 (ADDED) cierra las dos sugerencias del verify de P6 (`openspec/changes/archive/2026-11-12-spec-viva-openspec/verify-report.md`): una única definición de la etiqueta `Motivo:` y la paridad del marcador de «Sin impacto en la spec» entre el carril SDD (`isoevidence`) y el ODD (`livespec`).

## ADDED Requirements

### Requirement: Paridad del Marcador y del Motivo de «Sin impacto en la Spec» entre el Carril SDD y el Carril ODD (REQ-44.8)

El marcador `Sin impacto en la spec` y la etiqueta `Motivo:` DEBEN tener una única definición compartida (una constante exportada), usada tanto por la evidencia ISO del carril SDD (`ParseBugDeclaration`) como por la interpretación de la sección `## Impacto en la spec` del documento ODD (REQ-30.5). Ante las mismas líneas de la sección, ambas interpretaciones DEBEN coincidir en si el marcador está presente, en si hay una línea `Motivo:` con texto y en el valor del motivo: se ignoran solo los espacios iniciales y finales de cada línea, solo cuenta la primera línea `Motivo:` (si está vacía, el marcador queda incompleto aunque haya otra con texto) y la línea del marcador debe ser exacta. La paridad DEBE quedar fijada por una prueba tabular que pase las mismas entradas a ambas interpretaciones y compare marcador y motivo. Este requisito no cambia ninguna regla de REQ-30.5 ni de REQ-29.19: solo garantiza que no pueden divergir.

#### Scenario: Marcador y motivo completos

- **DADO** una sección con las líneas `Sin impacto en la spec` y `Motivo: solo cambia un mensaje de ayuda`
- **CUANDO** la interpretan `isoevidence` y `livespec`
- **ENTONCES** ambas informan marcador presente y el motivo `solo cambia un mensaje de ayuda`

#### Scenario: Marcador sin línea de motivo

- **DADO** una sección con la línea `Sin impacto en la spec` y sin ninguna línea `Motivo:`
- **CUANDO** la interpretan ambas
- **ENTONCES** ambas informan marcador presente y motivo ausente

#### Scenario: Motivo vacío

- **DADO** una sección con el marcador y la línea `Motivo:` sin texto
- **CUANDO** la interpretan ambas
- **ENTONCES** ambas informan marcador presente y motivo vacío (incompleto)

#### Scenario: Marcador ausente

- **DADO** una sección con `Motivo: algo` y sin la línea del marcador
- **CUANDO** la interpretan ambas
- **ENTONCES** ambas informan marcador ausente

#### Scenario: Espacios iniciales y finales

- **DADO** una sección con las líneas `  Sin impacto en la spec  ` y `   Motivo:   con espacios   `
- **CUANDO** la interpretan ambas
- **ENTONCES** ambas informan marcador presente y el motivo `con espacios`

#### Scenario: Solo cuenta la primera línea de motivo

- **DADO** una sección con el marcador, una primera línea `Motivo:` vacía y una segunda `Motivo: con texto`
- **CUANDO** la interpretan ambas
- **ENTONCES** ambas informan el motivo como vacío (incompleto)

#### Scenario: La etiqueta tiene una única definición

- **DADO** el árbol de código de producción tras este cambio
- **CUANDO** se busca la definición de la etiqueta `Motivo:`
- **ENTONCES** existe una sola constante exportada, usada por `isoevidence` y por `livespec`
- **Y** ninguno de los dos conserva un literal propio de la etiqueta

## MODIFIED Requirements

### Requirement: Comprobación de Colisión de Identificadores de Requisito según el Patrón Declarado (REQ-30.4)

Todo requisito ADDED de una delta ODD DEBE llevar en su título un identificador válido según el patrón declarado del proyecto (REQ-44.1 de `requirement-identifiers`; con el patrón por defecto, `(REQ-n.m)`), elegido por el autor (el siguiente número libre de la serie). La integración DEBE rechazar sin escribir un requisito ADDED cuyo identificador ya existe en `openspec/specs/` o en una delta activa de `openspec/changes/<cambio>/specs/` (la carpeta `archive/` queda excluida), así como dos requisitos ADDED con el mismo identificador dentro del propio documento. El motivo tipado DEBE nombrar el identificador, dónde colisiona (la capacidad o el cambio) y el siguiente número libre de la serie del identificador, calculada por el patrón (REQ-44.3 de `requirement-identifiers`): `REQ-31.2` tras `REQ-31.1` con el patrón por defecto, `RF-USU-02` tras `RF-USU-01` con `RF-<DOM>-<NN>`. Un requisito ADDED sin identificador válido según el patrón en el título DEBE rechazarse nombrando el requisito y el patrón en su texto legible (REQ-44.5). Un requisito que la spec canónica ya contiene con el mismo cuerpo no es una colisión (REQ-30.3).
(Previously: el identificador debía tener la forma literal `(REQ-n.m)` y el siguiente libre era el de la serie `n`; ahora lo fija el patrón declarado del proyecto, y con el patrón por defecto el comportamiento no cambia.)

Supuestos (propuesta §13, P5): el alcance de la comprobación es `openspec/specs/` y las deltas de los cambios activos, sin `archive/`; la obligación de que todo ADDED lleve identificador es una decisión de esta spec porque el indexador exige identificador (`living-documentation`, REQ-1.1).

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
- **DADO** un ADDED cuyo título no contiene un identificador válido (con el patrón por defecto, `(REQ-n.m)`)
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando el requisito y el patrón

#### Scenario: Una integración ya aplicada no colisiona consigo misma
- **DADO** un ADDED `(REQ-31.1)` ya integrado con el mismo cuerpo
- **CUANDO** se repite la integración
- **ENTONCES** no se reporta colisión y termina con código `0`

#### Scenario: Colisión con un patrón declarado
- **DADO** un proyecto con `sdd: {id_pattern: "RF-<DOM>-<NN>"}`, un `RF-USU-01` ya existente en `openspec/specs/usuarios/spec.md` y una delta ODD con un ADDED `(RF-USU-01)`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando `RF-USU-01`, la capacidad `usuarios` y el siguiente número libre de la serie `USU`, `RF-USU-02`
- **Y** no se escribe nada

#### Scenario: Un ADDED con el formato por defecto se rechaza con un patrón declarado
- **DADO** el mismo proyecto y un ADDED `(REQ-31.1)`
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza nombrando el requisito y el patrón `RF-<DOM>-<NN>`
- **Y** no se escribe nada

#### Scenario: El siguiente libre no cruza series
- **DADO** el mismo proyecto con `RF-USU-05` en `usuarios` y `RF-FAC-02` en `facturacion`, y un ADDED `(RF-FAC-02)` colisionando
- **CUANDO** se ejecuta la integración
- **ENTONCES** el siguiente libre propuesto es `RF-FAC-03`

## RENAMED Requirements

### Requirement: Comprobación de Colisión de Identificadores `REQ-n.m` (REQ-30.4) → Comprobación de Colisión de Identificadores de Requisito según el Patrón Declarado (REQ-30.4)

(Reason: el requisito deja de fijar la forma literal `REQ-n.m` y pide un identificador válido según el patrón que declara el proyecto (REQ-44.1 de `requirement-identifiers`); el título lo refleja)
(Migration: las referencias por el identificador REQ-30.4 siguen siendo válidas; las pruebas y textos que citaban el título antiguo pasan a citar el nuevo, y con el patrón por defecto el comportamiento observable no cambia)

#### Scenario: El siguiente libre rellena con el ancho del patrón y salta los usados del documento
- **DADO** el mismo proyecto con `RF-USU-01` en `usuarios`, una delta con un ADDED `(RF-USU-01)` colisionando y otro ADDED `(RF-USU-02)` en el mismo documento
- **CUANDO** se ejecuta la integración
- **ENTONCES** se rechaza proponiendo `RF-USU-03`, con el relleno de dos cifras del patrón
- **Y** no se escribe nada
