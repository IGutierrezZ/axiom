# Especificación de Requerimientos: Contexto Técnico del Proyecto: Declaración, Lectura y Mantenimiento (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Rebanadas:** B (declaración, proyección y lectura) y C (delta de contexto) · **Tipo:** capacidad nueva
> **Idioma:** Español (castellano peninsular)

## Propósito

Definir el contexto técnico del proyecto como concepto de primera clase: cómo se declara en `axiom.yaml` con dos modos (**mantenido** y **de referencia**), cómo lo leen las fases SDD, cómo lo mantiene el flujo mediante un **delta de contexto** por cambio, simétrico a las delta specs, y la garantía de que la documentación de referencia nunca se modifica.

**Resultado esperado:** cuando un cambio altera algo que documenta el contexto mantenido, el flujo SDD lo actualiza con traza por entrada; la documentación de referencia permanece intacta; y un cambio nunca omite el impacto en silencio.

Esta capacidad se completa con cuatro especificaciones vecinas:

| Dónde | Qué fija |
|---|---|
| `sdd-archive-lifecycle` (REQ-21.17 y REQ-24.55 a REQ-24.57) | Integración del delta en archive, traza, creación de documentos mantenidos y paradas |
| `spec-baseline-declaration` (REQ-24.62) | Almacén de artefactos de ficheros que exige el contexto mantenido |
| `sdd-block-review-gates` (REQ-21.9 y REQ-21.11) | Qué evalúan las compuertas `design` y `role-apply` |
| `organic-agent-trigger-rules` (REQ-24.58) | Regla mínima del carril ODD, que no tiene archive |

## Términos

| Término | Definición |
|---|---|
| Contexto técnico | Documentación técnica del proyecto (arquitectura, componentes, comandos de build, convenciones) que las fases leen como entrada. |
| Contexto mantenido | Contexto técnico vivo cuyo mantenimiento corresponde al flujo SDD: lo actualiza archive a partir del delta de contexto. |
| Contexto de referencia | Documentación que solo se usa como entrada: ninguna fase la modifica jamás. |
| Entrada | Fichero que las fases lectoras leen por defecto en cada modo. |
| Delta de contexto | Fichero del cambio que lista qué partes del contexto mantenido cambian y por qué, o declara «Sin impacto». |
| Ancla | Referencia a una sección de un documento por fichero y sección. |

## Ejemplo de declaración (no normativo)

```yaml
technical_context:
  maintained:
    entry: context/TECHNICAL_CONTEXT.md
    paths:
      - context/
  reference:
    entry: 00_Resumen_Ejecutivo.md
    paths:
      - 00_Glosario.md
      - 02_Requisitos_No_Funcionales.md
```

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| REQ-24.26 | Bloque `technical_context:` con dos modos | `maintained` y `reference`, sin rutas en ambos |
| REQ-24.27 | Validación del contexto técnico | Loader y `axiom workspace validate`; ruta inexistente solo avisa; el almacén sin ficheros se reporta (REQ-24.62) |
| REQ-24.28 | Proyección de `technicalContext` | Aditiva, sin tocar el estado de artefactos |
| REQ-24.29 | Lectura por las fases lectoras | Entrada por defecto y resto bajo demanda |
| REQ-24.30 | Las rutas de referencia nunca se modifican | Ninguna fase ni la integración de archive las escribe |
| REQ-24.31 | Delta de contexto por cambio | Entradas o «Sin impacto» explícito |
| REQ-24.32 | Anclaje por fichero y sección | Tolera títulos repetidos |
| REQ-24.33 | Redacción del impacto previsto | `sdd-design` redacta; no escribe el contexto |
| REQ-24.34 | Actualización del impacto final | `sdd-apply` actualiza el delta, no `design.md` |
| REQ-24.35 | Aviso no bloqueante de verify | Heurístico e informativo |
| REQ-24.36 | La ausencia del delta nunca se asume | Se señala y archive se detiene |
| REQ-24.37 | Rutas mantenidas fuera del repositorio de especificaciones | Autorización por cambio, nunca implícita |

## No cubre esta capacidad

- Generar un contexto técnico inicial a partir del código (arranque en frío del contexto).
- Indexar el contexto técnico en los catálogos de livingdoc, dashboard o knowledge.
- El mantenimiento del contexto en el carril ODD más allá de la regla mínima de REQ-24.58.

---

## 1. Capacidad: `project-technical-context`

El contexto técnico se declara una vez en `axiom.yaml`, se proyecta como localizador en el status y se mantiene por cambio con un delta que archive integra.

### Requirement: Bloque `technical_context:` con Dos Modos (REQ-24.26)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `technical_context:` con los modos `maintained` (contexto técnico vivo que el flujo SDD mantiene al día) y `reference` (documentación que solo se lee y nunca se modifica). Cada modo declarado DEBE incluir `paths` (lista no vacía de ficheros o directorios) y PUEDE incluir `entry` (fichero que las fases lectoras leen por defecto, que DEBE estar contenido en `paths` del mismo modo). Las rutas son relativas a `workspaceRoot`, se admiten `/` y `\` como separadores, DEBEN permanecer dentro de la carpeta maestra y NO PUEDEN ser absolutas. Una ruta NO DEBE pertenecer a ambos modos; dos rutas solapan cuando son iguales o una contiene a la otra. Sin el bloque no hay contexto técnico declarado.
(Previously: una ruta tampoco podía pertenecer a `baseline.documents`; la baseline ajena se retira)

#### Scenario: Declaración con los dos modos
- **DADO** un `axiom.yaml` con `maintained` (`entry: context/TECHNICAL_CONTEXT.md`, `paths: [context/]`) y `reference` (`paths: [00_Glosario.md]`)
- **CUANDO** se carga y se valida
- **ENTONCES** la declaración es válida

#### Scenario: Solo contexto de referencia
- **DADO** un `axiom.yaml` con `reference` y sin `maintained`
- **CUANDO** se carga y se valida
- **ENTONCES** la declaración es válida
- **Y** ningún documento queda sujeto a mantenimiento por archive
- **Y** la spec viva sigue siendo `openspec/specs/`

#### Scenario: Ruta en ambos modos
- **DADO** una ruta `context/a.md` en `maintained.paths` y la misma ruta en `reference.paths`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta un error que nombra la ruta y los dos modos

#### Scenario: Solapamiento por contención
- **DADO** `context/` en `maintained.paths` y `context/a.md` en `reference.paths`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta un error que nombra ambas rutas

#### Scenario: Entrada fuera de las rutas del modo
- **DADO** un modo con `entry: README.md` y `paths: [context/]`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta que `entry` debe estar contenida en `paths`

#### Scenario: Ruta absoluta o fuera de la carpeta maestra
- **DADO** una ruta absoluta y otra `../../fuera/` que sale de la carpeta maestra
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta un error por cada una nombrando la regla

---

### Requirement: Validación del Contexto Técnico (REQ-24.27)

La sintaxis (modos admitidos, claves desconocidas, `paths` no vacío, rutas relativas y contenidas, `entry` contenida en `paths` y solapes entre modos) DEBE validarse al cargar `axiom.yaml`. Una ruta declarada que no existe NO DEBE ser un error: DEBE reportarse como advertencia sin alterar el código de salida, porque un documento mantenido puede crearse en archive (REQ-24.55). Toda violación DEBE reportarse en español nombrando la ruta y la regla, y `axiom workspace validate` DEBE finalizar con código de salida `1` si hay alguna. Con `maintained` declarado, `axiom workspace validate` DEBE reportar además como error la incompatibilidad con un almacén de artefactos sin ficheros (`engram` o `none`), según REQ-30.19.
(Previously: validaba también los solapes con `baseline.documents` y citaba REQ-24.62 de `spec-baseline-declaration`)

#### Scenario: Ruta inexistente solo avisa
- **DADO** un modo `maintained` con `paths: [context/arquitectura.md]` y ese fichero inexistente
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta una advertencia que nombra la ruta
- **Y** el resultado es `COMPLIANT` y el código de salida es `0`

#### Scenario: Lista de rutas vacía
- **DADO** un modo con `paths: []`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español que el modo requiere al menos una ruta
- **Y** el código de salida es `1`

#### Scenario: Modo desconocido
- **DADO** un bloque `technical_context:` con una clave `readonly:` en lugar de `reference:`
- **CUANDO** se carga `axiom.yaml`
- **ENTONCES** la carga falla nombrando la clave desconocida y los modos admitidos
- **Y** `axiom workspace validate` finaliza con código `1`

---

### Requirement: Proyección Aditiva de `technicalContext` en el Status v2 (REQ-24.28)

Cuando `axiom.yaml` declara `technical_context:` y la declaración es válida, el status v2 DEBE incluir el campo opcional `technicalContext` con, por cada modo declarado (`maintained`, `reference`), `paths` (rutas absolutas) y, si se declaró, `entry` (ruta absoluta). Un modo no declarado NO DEBE aparecer y, sin declaración, el campo NO DEBE aparecer. El campo se proyecta con o sin cambio seleccionado y NO DEBE alterar `artifacts`, `artifactPaths` ni `contextFiles`: el delta de contexto no se proyecta en el status.

#### Scenario: Ambos modos proyectados
- **DADO** un `technical_context:` con los dos modos
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `technicalContext` contiene `maintained` y `reference`, cada uno con `paths` absolutos y su `entry` absoluta

#### Scenario: Un solo modo
- **DADO** un `technical_context:` solo con `reference`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `technicalContext` contiene únicamente `reference`

#### Scenario: Sin declaración
- **DADO** un `axiom.yaml` sin `technical_context:`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el JSON no contiene `technicalContext`

#### Scenario: El estado de artefactos no cambia
- **DADO** un cambio con delta de contexto presente y `technical_context.maintained` declarado
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** las claves de `artifacts` y de `artifactPaths` son las mismas que sin delta de contexto

---

### Requirement: Lectura del Contexto por las Fases Lectoras y Presupuesto de Lectura (REQ-24.29)

Cuando el status proyecta `technicalContext`, las skills `sdd-explore`, `sdd-design`, `sdd-apply` y `sdd-verify` DEBEN leer el fichero `entry` de cada modo declarado y DEBEN consultar el resto de `paths` solo bajo demanda, sin cargarlo por anticipado. Las demás fases NO están obligadas a leerlo. Las skills DEBEN obtener los localizadores del status y NO DEBEN interpretar `axiom.yaml`.

#### Scenario: Las cuatro fases lectoras leen la entrada
- **DADO** los textos de `sdd-explore`, `sdd-design`, `sdd-apply` y `sdd-verify`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** cada uno indica leer la `entry` de cada modo de `technicalContext` y consultar el resto bajo demanda

#### Scenario: Modo sin entrada
- **DADO** un modo declarado sin `entry`
- **CUANDO** una fase lectora prepara su entrada
- **ENTONCES** no precarga ningún fichero de ese modo y solo lo consulta cuando lo necesita

#### Scenario: Sin contexto declarado
- **DADO** un status sin `technicalContext`
- **CUANDO** se ejecutan las fases
- **ENTONCES** operan como antes de esta capacidad

---

### Requirement: Las Rutas de Referencia Nunca se Modifican (REQ-24.30)

Ninguna fase del flujo SDD (explore, propose, spec, design, tasks, apply, verify y archive) NI la integración de archive NI la integración ODD (`axiom archive integrate --odd`) DEBE modificar, crear ni eliminar ficheros bajo las rutas del modo `reference`. El texto de cada skill de fase DEBE prescribirlo, la verificación nativa de la integración del contexto (REQ-30.17) DEBE rechazar cualquier diff que las afecte y la integración ODD DEBE rechazar una rama que las modifique (REQ-30.6 de `odd-living-spec-integration`).
(Previously: citaba la verificación nativa de REQ-24.51 y no mencionaba la integración ODD)

#### Scenario: El archivado no deja cambios en referencia
- **DADO** un contexto de referencia y un cambio con delta de contexto sobre rutas mantenidas
- **CUANDO** archive integra el cambio
- **ENTONCES** el diff del repositorio no contiene ningún cambio bajo las rutas de referencia

#### Scenario: Una entrada del delta dirigida a una ruta de referencia
- **DADO** un delta de contexto con una entrada cuyo documento está bajo `reference.paths`
- **CUANDO** se valida el delta
- **ENTONCES** se rechaza nombrando la entrada y el modo `reference`

#### Scenario: Una edición que toca una ruta de referencia
- **DADO** una edición de archive cuyo diff incluye un fichero bajo `reference.paths`
- **CUANDO** se ejecuta la verificación nativa
- **ENTONCES** falla nombrando el fichero
- **Y** archive se detiene sin mover el cambio a `openspec/changes/archive/`

#### Scenario: La integración ODD rechaza una rama que toca la referencia
- **DADO** una rama ODD que modifica un fichero bajo `reference.paths`
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** se rechaza nombrando el fichero

#### Scenario: Las skills lo prescriben
- **DADO** los textos de las skills de las fases del flujo SDD
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** cada uno prohíbe escribir en las rutas del modo `reference`
### Requirement: Delta de Contexto por Cambio (REQ-24.31)

Cuando `technical_context.maintained` está declarado, cada cambio DEBE llevar un delta de contexto: un fichero propio del cambio, en su carpeta, sin proyección en el status ni en el estado de artefactos. Su nombre de trabajo es `context-impact.md`; el diseño PUEDE fijar otro sin alterar estos requisitos. El delta DEBE contener o bien una o más entradas o bien una declaración explícita «Sin impacto» con su motivo, y NO DEBE mezclar ambas. Cada entrada DEBE indicar el documento (dentro de las rutas del modo `maintained`), el ancla (REQ-24.32), la acción (`add`, `modify` o `remove`) y el motivo del cambio. «Sin impacto» queda registrado como decisión del cambio. El formato concreto del fichero lo fija el diseño y DEBE poder analizarse de forma determinista. Sin `technical_context.maintained`, el delta NO es exigible.

*Ejemplo no normativo:* una entrada puede expresarse como `context/ARQUITECTURA.md`, ancla `Componentes > Servicio de pagos`, acción `modify`, motivo «se añade el reintento con retroceso exponencial»; la alternativa es una única línea `Sin impacto: el cambio solo corrige una errata en un mensaje de error`.

#### Scenario: Delta con entradas válidas
- **DADO** `technical_context.maintained` con `paths: [context/]` y un delta con una entrada `modify` sobre `context/ARQUITECTURA.md` con ancla y motivo
- **CUANDO** se valida el delta
- **ENTONCES** el delta es válido

#### Scenario: «Sin impacto» con motivo
- **DADO** un delta que solo declara «Sin impacto» con su motivo
- **CUANDO** se valida el delta
- **ENTONCES** el delta es válido y queda registrado como decisión del cambio

#### Scenario: «Sin impacto» sin motivo
- **DADO** un delta que declara «Sin impacto» sin motivo
- **CUANDO** se valida el delta
- **ENTONCES** se rechaza indicando que la decisión requiere un motivo

#### Scenario: Mezcla de entradas y «Sin impacto»
- **DADO** un delta con una entrada y además «Sin impacto»
- **CUANDO** se valida el delta
- **ENTONCES** se rechaza indicando que no pueden coexistir

#### Scenario: Entrada fuera de las rutas mantenidas
- **DADO** una entrada cuyo documento no está bajo `maintained.paths`
- **CUANDO** se valida el delta
- **ENTONCES** se rechaza nombrando el documento y las rutas mantenidas declaradas

#### Scenario: Sin contexto mantenido el delta no se exige
- **DADO** un `axiom.yaml` con solo `reference` o sin `technical_context:`
- **CUANDO** un cambio llega a archive sin delta de contexto
- **ENTONCES** archive no exige el delta

---

### Requirement: Anclaje por Fichero y Sección con Títulos Repetidos (REQ-24.32)

Cada entrada del delta se ancla por fichero y sección. Cuando el título de la sección se repite dentro del fichero, la entrada DEBE desambiguarlo mediante la ruta de encabezados ancestros o un ordinal de aparición. Un ancla que resuelve a más de una sección DEBE rechazarse, y un ancla que no resuelve a ninguna sección solo es válida para la acción `add`.

#### Scenario: Título repetido desambiguado por la ruta de encabezados
- **DADO** un fichero con dos secciones tituladas «Notas», una bajo «Pagos» y otra bajo «Envíos»
- **CUANDO** una entrada ancla `Pagos > Notas`
- **ENTONCES** el ancla resuelve a una única sección

#### Scenario: Ancla ambigua
- **DADO** un fichero con dos secciones «Notas» bajo el mismo encabezado padre
- **CUANDO** una entrada ancla `Pagos > Notas` sin ordinal
- **ENTONCES** se rechaza nombrando las dos apariciones (línea y encabezado)

#### Scenario: Ancla inexistente según la acción
- **DADO** un ancla que no resuelve a ninguna sección
- **CUANDO** la acción es `modify` o `remove`
- **ENTONCES** la entrada se rechaza
- **Y** con la acción `add` la entrada es válida

---

### Requirement: Redacción del Impacto Previsto en `sdd-design` (REQ-24.33)

Con `technical_context.maintained` declarado, la skill `sdd-design` DEBE redactar el delta de contexto con el impacto previsto del diseño (o «Sin impacto» razonado) y NO DEBE modificar ningún documento del contexto. La compuerta `design` revisa el delta (REQ-21.9).

#### Scenario: El diseño genera el delta
- **DADO** `technical_context.maintained` declarado y un diseño que añade un módulo nuevo
- **CUANDO** se ejecuta `sdd-design`
- **ENTONCES** existe el delta de contexto en la carpeta del cambio con el impacto previsto o con «Sin impacto» razonado

#### Scenario: El diseño no escribe en el contexto
- **DADO** el mismo cambio
- **CUANDO** termina `sdd-design`
- **ENTONCES** el diff del repositorio no contiene cambios bajo `maintained.paths`

#### Scenario: Sin contexto mantenido no se genera delta
- **DADO** un `axiom.yaml` sin `technical_context.maintained`
- **CUANDO** se ejecuta `sdd-design`
- **ENTONCES** no se crea ningún delta de contexto

---

### Requirement: Actualización del Impacto Final en `sdd-apply` (REQ-24.34)

Con `technical_context.maintained` declarado, la skill `sdd-apply` DEBE actualizar el delta de contexto para reflejar lo realmente implementado (añadir, modificar o retirar entradas, o cambiar entre entradas y «Sin impacto» con motivo) sin reabrir `design.md` y sin modificar ningún documento del contexto. En un cambio con varios roles, cada rol DEBE actualizar el mismo delta del cambio. La compuerta `role-apply` revisa el impacto final (REQ-21.11).

#### Scenario: La implementación revela un impacto no previsto
- **DADO** un delta con «Sin impacto» y una implementación que cambia un comando de build documentado en el contexto mantenido
- **CUANDO** termina `sdd-apply`
- **ENTONCES** el delta contiene una entrada `modify` para el documento afectado y ya no declara «Sin impacto»

#### Scenario: Apply no reabre el diseño ni escribe el contexto
- **DADO** un cambio con `design.md` aprobado
- **CUANDO** se ejecuta `sdd-apply`
- **ENTONCES** `design.md` conserva sus bytes y el diff no contiene cambios bajo `maintained.paths`

#### Scenario: Varios roles actualizan el mismo delta
- **DADO** un cambio con los roles `core` y `web`
- **CUANDO** cada rol ejecuta `sdd-apply`
- **ENTONCES** ambos actualizan el mismo delta de contexto del cambio y no se crea uno por rol

---

### Requirement: Aviso No Bloqueante de `sdd-verify` ante un «Sin Impacto» Dudoso (REQ-24.35)

Con `technical_context.maintained` declarado y un delta que declara «Sin impacto», `sdd-verify` DEBE emitir un aviso informativo cuando el diff del cambio toque: (a) un manifiesto de dependencias (por ejemplo `go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, `requirements.txt` o `*.csproj`), (b) un módulo nuevo de primer nivel (un directorio de primer nivel creado por el cambio) o (c) ficheros de comandos de build o CI (por ejemplo `Makefile`, `Dockerfile`, `.github/workflows/*` o `.goreleaser.yaml`). El aviso DEBE nombrar las rutas que lo disparan y constar en una sección de avisos de `verify-report.md`, y NUNCA DEBE bloquear el archivado ni alterar `nextRecommended` ni `dependencies`. Sin esas señales, o con entradas en el delta, el aviso NO se emite.

#### Scenario: Manifiesto de dependencias con «Sin impacto»
- **DADO** un delta con «Sin impacto» y un diff que modifica `go.mod`
- **CUANDO** se ejecuta `sdd-verify`
- **ENTONCES** `verify-report.md` contiene un aviso que nombra `go.mod`
- **Y** `dependencies.archive` no pasa a `blocked` por ese aviso

#### Scenario: Módulo nuevo de primer nivel
- **DADO** un delta con «Sin impacto» y un diff que crea el directorio `payments/` en la raíz
- **CUANDO** se ejecuta `sdd-verify`
- **ENTONCES** el aviso nombra `payments/`

#### Scenario: Sin señales no hay aviso
- **DADO** un delta con «Sin impacto» y un diff que solo modifica ficheros de código dentro de módulos existentes
- **CUANDO** se ejecuta `sdd-verify`
- **ENTONCES** no se emite ningún aviso de contexto

#### Scenario: Con entradas en el delta no hay aviso
- **DADO** un delta con una entrada `modify` y un diff que modifica `go.mod`
- **CUANDO** se ejecuta `sdd-verify`
- **ENTONCES** no se emite aviso de contexto

---

### Requirement: La Ausencia del Delta de Contexto Nunca se Asume (REQ-24.36)

Con `technical_context.maintained` declarado y sin delta de contexto en el cambio, el sistema NO DEBE asumir «Sin impacto»: la compuerta `design` DEBE señalar la ausencia (REQ-21.9) y el archivado DEBE detenerse pidiendo registrar el impacto o «Sin impacto» (REQ-24.56). Esto DEBE cumplirse también cuando el cambio se sella en modalidad continua, donde no hay compuertas.

#### Scenario: La compuerta de diseño señala la ausencia
- **DADO** un cambio en modalidad con paradas sin delta de contexto y `technical_context.maintained` declarado
- **CUANDO** se presenta la compuerta `design`
- **ENTONCES** la presentación señala explícitamente que falta el delta de contexto

#### Scenario: Archive se detiene sin el delta
- **DADO** el mismo cambio, en modalidad con paradas o continua, que llega a archive sin delta
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene pidiendo registrar el impacto o «Sin impacto»
- **Y** el cambio permanece en `openspec/changes/`

---

### Requirement: Rutas Mantenidas Fuera del Repositorio de Especificaciones (REQ-24.37)

Una ruta del modo `maintained` situada fuera del repositorio de especificaciones (por ejemplo, documentación por rol en un repositorio de código hermano) PUEDE declararse. Su integración DEBE exigir la autorización de edición por cambio existente (`axiom sdd-attempt grant`) y NUNCA se concede de forma implícita. La verificación por diff de archive DEBE cubrir cada repositorio afectado.

#### Scenario: Sin autorización archive se detiene
- **DADO** una ruta mantenida `../repo-backend/docs/` y un delta con una entrada sobre ese directorio, sin autorización de edición para `repo-backend`
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene nombrando la autorización de edición que falta
- **Y** no modifica ningún documento

#### Scenario: Con autorización la integración se verifica en ambos repositorios
- **DADO** la misma situación con la autorización concedida para `repo-backend`
- **CUANDO** archive integra el delta
- **ENTONCES** la verificación por diff comprueba el repositorio de especificaciones y `repo-backend`

#### Scenario: Los repositorios de código no se autorizan por declarar la ruta
- **DADO** una ruta mantenida en `repo-backend`
- **CUANDO** se consulta el status
- **ENTONCES** `actionContext.allowedEditRoots` no incluye `repo-backend`
### Requirement: Superficie CLI del Contexto Técnico en `axiom archive context` (REQ-30.15)

El grupo `axiom archive` DEBE exponer `axiom archive context check|prepare|verify|show`, con el mismo contrato observable que tenía la parte de contexto de `axiom sdd integrate` y restringido al contexto técnico mantenido. `check` DEBE ser de solo lectura: evalúa el delta de contexto del cambio (REQ-24.31, REQ-24.32 y REQ-24.36) y informa de los avisos y de los bloqueos, sin escribir ni exigir git ni autorización. `prepare` DEBE dejar en la carpeta del cambio únicamente el plan de la integración del contexto. `verify` DEBE ejecutar la verificación nativa (REQ-30.17) y dejar únicamente la traza (REQ-30.18). `show` DEBE leer la traza y presentarla en `markdown` o `json`. Cada verbo DEBE finalizar con código distinto de cero cuando informa de algún bloqueo. Los verbos NO DEBEN operar sobre dominios de la spec viva ni sobre ningún documento fuera de las rutas `maintained`. La orden `axiom sdd integrate` deja de existir (REQ-30.25 de `sdd-archive-lifecycle`).

Supuesto (propuesta §13, P1): el nombre de la superficie es `axiom archive context check|prepare|verify|show`; los nombres de los ficheros de plan y traza los fija el diseño.

#### Scenario: `check` no escribe
- **DADO** un cambio con `technical_context.maintained` declarado y un delta de contexto válido
- **CUANDO** se ejecuta `axiom archive context check --change <cambio>`
- **ENTONCES** informa del estado del delta de contexto
- **Y** no se crea ni se modifica ningún fichero

#### Scenario: `check` bloquea ante la ausencia del delta
- **DADO** un contexto mantenido declarado y un cambio sin delta de contexto
- **CUANDO** se ejecuta `axiom archive context check --change <cambio>`
- **ENTONCES** informa del bloqueo pidiendo registrar el impacto o «Sin impacto»
- **Y** el código de salida es distinto de cero

#### Scenario: `prepare` y `verify` escriben solo en la carpeta del cambio
- **DADO** un cambio con delta de contexto válido
- **CUANDO** se ejecutan `prepare` y `verify`
- **ENTONCES** los únicos ficheros nuevos están en la carpeta del cambio (plan y traza)

#### Scenario: `show` presenta la traza
- **DADO** una traza existente
- **CUANDO** se ejecuta `axiom archive context show --change <cambio> --format json`
- **ENTONCES** la salida es el contenido de la traza en JSON
- **Y** no se escribe nada

#### Scenario: La ayuda del grupo lista los verbos
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom archive --help`
- **ENTONCES** la ayuda lista `context` con sus cuatro verbos

---

### Requirement: Integración Dirigida del Contexto Mantenido (REQ-30.16)

Archive DEBE integrar el delta de contexto en las rutas del modo `maintained` mediante una edición dirigida por agente: por cada entrada del delta (acciones `add`, `modify` o `remove`), el agente DEBE crear o editar el documento y la sección ancla, y la entrada DEBE quedar en la traza (REQ-30.18). El agente DEBE editar únicamente los documentos nombrados en las entradas del delta y que estén bajo las rutas `maintained`, y DEBE reutilizar la sección ya editada si existe, de modo que una integración repetida no duplique secciones. (Este requisito es REQ-24.50 de `spec-baseline-integration`, restringido al contexto técnico.)

#### Scenario: Entrada `modify` integrada
- **DADO** un delta con una entrada `modify` sobre `context/ARQUITECTURA.md`, ancla `Componentes > Servicio de pagos`
- **CUANDO** archive integra el cambio
- **ENTONCES** la sección ancla refleja el cambio descrito
- **Y** la traza contiene una entrada para ella

#### Scenario: Integración repetida
- **DADO** un documento que ya contiene la sección editada
- **CUANDO** se repite la integración
- **ENTONCES** se edita la sección existente y el documento no contiene secciones duplicadas

---

### Requirement: Verificación Nativa de la Integración del Contexto (REQ-30.17)

Tras la edición, una verificación nativa y determinista (sin modelo) DEBE comprobar: (a) presencia: cada entrada del delta de contexto figura en la traza; (b) anclas: cada ancla citada existe tras la edición exactamente una vez, salvo en la acción `remove`, donde ya no existe; (c) alcance: el diff de cada repositorio afectado solo toca los documentos nombrados en las entradas del delta de contexto y los ficheros de la propia carpeta del cambio, y ninguna ruta del modo `reference`; con una decisión «Sin impacto», el diff en las rutas `maintained` es vacío (REQ-24.57); y (d) la traza existe y es analizable. Si alguna comprobación falla, la verificación DEBE finalizar con código distinto de cero nombrando la entrada y la comprobación, y archive DEBE detenerse sin mover el cambio a `openspec/changes/archive/`. La verificación NO evalúa la corrección semántica de la edición. (Este requisito es REQ-24.51 de `spec-baseline-integration`, restringido al contexto técnico y sin la comprobación de sustitución de identificadores provisionales, que deja de existir con REQ-24.55.)

#### Scenario: Verificación superada
- **DADO** una edición con todas las entradas en la traza, anclas únicas y un diff limitado a los documentos nombrados
- **CUANDO** se ejecuta la verificación
- **ENTONCES** finaliza con código `0`

#### Scenario: Entrada ausente de la traza
- **DADO** una entrada del delta sin entrada en la traza
- **CUANDO** se ejecuta la verificación
- **ENTONCES** falla nombrando la entrada y la comprobación de presencia

#### Scenario: Ancla inexistente o repetida
- **DADO** una entrada cuya ancla no existe tras la edición, o existe dos veces
- **CUANDO** se ejecuta la verificación
- **ENTONCES** falla nombrando la entrada y el ancla

#### Scenario: El diff toca un documento no nombrado
- **DADO** una edición que modifica un documento de `maintained.paths` que ninguna entrada nombra
- **CUANDO** se ejecuta la verificación
- **ENTONCES** falla nombrando el fichero y la comprobación de alcance

#### Scenario: El diff toca una ruta de referencia
- **DADO** una edición que modifica un fichero bajo `reference.paths`
- **CUANDO** se ejecuta la verificación
- **ENTONCES** falla nombrando el fichero

#### Scenario: Una edición contradice «Sin impacto»
- **DADO** un delta «Sin impacto» y una edición que modifica un documento de `maintained.paths`
- **CUANDO** se ejecuta la verificación
- **ENTONCES** falla nombrando el fichero y la contradicción con la decisión registrada
- **Y** archive se detiene sin mover el cambio a `openspec/changes/archive/`

#### Scenario: Tras un fallo el cambio permanece activo
- **DADO** una verificación fallida
- **CUANDO** archive se detiene
- **ENTONCES** el cambio permanece en `openspec/changes/` y no existe su carpeta bajo `openspec/changes/archive/`

---

### Requirement: Traza de Integración del Contexto por Entrada (REQ-30.18)

Archive DEBE registrar una traza por entrada del delta de contexto, en un fichero dentro de la carpeta del cambio (que se archiva con él; el nombre lo fija el diseño) y en una sección del informe de archive. Cada entrada DEBE incluir el documento, el ancla, la acción (`added`, `modified`, `removed` o `created`) y el estado (`integrated` o `pending`). Para «Sin impacto», la traza DEBE registrar la decisión y su motivo. (Este requisito es REQ-24.52 de `spec-baseline-integration`, restringido al contexto técnico y sin el mapeo de identificadores provisionales.)

#### Scenario: Traza en la carpeta del cambio y en el informe
- **DADO** un archivado que integra dos entradas de contexto
- **CUANDO** termina la integración
- **ENTONCES** la carpeta del cambio contiene la traza con dos entradas
- **Y** el informe de archive contiene una sección con las mismas dos entradas

#### Scenario: Entrada creada
- **DADO** una entrada `add` sobre un documento mantenido que no existía
- **CUANDO** se inspecciona la traza
- **ENTONCES** la entrada registra la acción `created` y el estado `integrated`

#### Scenario: Decisión «Sin impacto»
- **DADO** un delta de contexto con «Sin impacto» y su motivo
- **CUANDO** archive lo registra
- **ENTONCES** la traza y el informe contienen la decisión y el motivo

---

### Requirement: El Contexto Técnico Mantenido Exige un Almacén de Artefactos de Ficheros (REQ-30.19)

Un contexto técnico mantenido (`technical_context.maintained`) DEBE usarse con un almacén de ficheros (`openspec` o `hybrid`), porque la integración de archive lee y escribe ficheros de la carpeta del cambio: el delta de contexto (REQ-24.31) y la traza (REQ-30.18). El almacén es el que declara `artifact_store` en `openspec/config.yaml` y, si no se declara, el que resuelve el status para el cambio. Con `engram` o `none`:

- `axiom workspace validate` DEBE reportar un error en español que nombre `technical_context.maintained`, el almacén declarado y la continuación (declarar `artifact_store: openspec` o `hybrid`, o retirar la declaración), y finalizar con código de salida `1`. La validación solo evalúa el valor declarado: sin `artifact_store` declarado no reporta nada por este motivo.
- El status DEBE marcar el archivado como bloqueado (`dependencies.archive` en `blocked`) con esa causa y la continuación. Las dependencias de las fases de planificación y de `apply` NO DEBEN pasar a `blocked` por ella y el status sigue proyectando `technicalContext`.
- Archive DEBE detenerse (REQ-24.56, caso e).

Una declaración solo de contexto de referencia (`technical_context.reference`) y la ausencia de contexto mantenido NO exigen un almacén de ficheros. (Este requisito es la parte de contexto de REQ-24.62 de `spec-baseline-declaration`.)

#### Scenario: Contexto mantenido con almacén `none`
- **DADO** un `axiom.yaml` con `technical_context.maintained` válido y un `openspec/config.yaml` con `artifact_store: none`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español que el contexto mantenido exige un almacén de ficheros, nombrando `technical_context.maintained`, `none` y la continuación
- **Y** el código de salida es `1`

#### Scenario: Los almacenes de ficheros y la ausencia de declaración no se reportan
- **DADO** la misma declaración con `artifact_store: openspec`, con `hybrid` y sin `artifact_store` declarado
- **CUANDO** se ejecuta `axiom workspace validate` en cada caso
- **ENTONCES** ninguno reporta error por el almacén

#### Scenario: El status bloquea el archivado sin bloquear la planificación
- **DADO** un contexto mantenido, `artifact_store: engram` declarado y un cambio resuelto en ese almacén
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `dependencies.archive` es `blocked` con una causa que nombra el almacén `engram` y la continuación
- **Y** `technicalContext` sigue proyectado y las dependencias de planificación y de `apply` no son `blocked` por esa causa

#### Scenario: Sin contexto mantenido el almacén no se exige
- **DADO** un `axiom.yaml` sin contexto mantenido (o con solo `technical_context.reference`) y `artifact_store: engram`
- **CUANDO** se ejecutan `axiom workspace validate` y `axiom sdd status --json`
- **ENTONCES** no se reporta error por el almacén y `dependencies.archive` no se bloquea por esa causa

---

### Requirement: El Mantenimiento del Contexto en ODD se Rige por `odd-living-spec-integration` (REQ-30.20)

El mantenimiento del contexto técnico en el carril ODD DEBE ser el de la regla mínima de cierre de `organic-agent-trigger-rules` (REQ-24.58) más la comprobación estructural de `axiom archive integrate --odd` (REQ-30.6 de `odd-living-spec-integration`): rechazo de cambios en rutas `reference` e informe de las rutas `maintained` tocadas, sin marcador de impacto. Esta capacidad NO DEBE exigir en ODD ningún delta de contexto, traza ni archive. (Sustituye a la exclusión «El mantenimiento del contexto en el carril ODD más allá de la regla mínima de REQ-24.58» del apartado «No cubre» vigente.)

#### Scenario: ODD no genera delta de contexto
- **DADO** un trabajo ODD con contexto mantenido declarado
- **CUANDO** se cierra la feature
- **ENTONCES** no se crea ningún delta de contexto ni traza de integración

#### Scenario: La comprobación estructural es la de la integración ODD
- **DADO** una rama ODD que modifica una ruta de `reference`
- **CUANDO** se ejecuta `axiom archive integrate --odd <feature>`
- **ENTONCES** se rechaza según REQ-30.6

---

### Requirement: Precondiciones de la Integración del Contexto (REQ-30.29)

Antes de modificar ninguna ruta mantenida, archive DEBE comprobar que (a) cada repositorio que contiene un documento afectado es un repositorio git y (b) los documentos afectados no tienen cambios sin confirmar ni están sin seguimiento. Si alguna comprobación falla, archive DEBE detenerse sin modificar nada, nombrando los ficheros afectados y la continuación (confirmar, guardar o descartar los cambios, o versionar el repositorio), y el cambio permanece en `openspec/changes/`. Los documentos que archive crearía (REQ-24.55) quedan excluidos de (b). Archive NO DEBE actualizar `REGISTRO_INCREMENTOS.md`, `REGISTRO_BUGS.md` ni ningún documento no nombrado en el delta de contexto. (Es REQ-24.49 de `spec-baseline-integration`, restringido al contexto técnico.)

Supuesto: REQ-24.49 no figura en la tabla de reubicaciones de la propuesta (§4.3); se reubica aquí porque su parte de contexto sigue vigente.

#### Scenario: Documento afectado con cambios sin confirmar
- **DADO** un documento mantenido afectado por el delta con modificaciones sin confirmar
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene nombrando el fichero y la continuación
- **Y** ningún documento se modifica

#### Scenario: Cambios sin confirmar en documentos no afectados
- **DADO** un documento mantenido que el delta no nombra con modificaciones sin confirmar
- **CUANDO** se solicita el archivado
- **ENTONCES** archive no se detiene por ese documento

#### Scenario: Repositorio que no es git
- **DADO** un documento afectado dentro de un directorio que no es un repositorio git
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene indicando que la verificación por diff requiere un repositorio git y cómo versionarlo

#### Scenario: Los registros del proyecto no se tocan
- **DADO** un proyecto con `REGISTRO_INCREMENTOS.md` y `REGISTRO_BUGS.md` no nombrados en el delta
- **CUANDO** archive integra un cambio
- **ENTONCES** el diff no contiene cambios en esos ficheros

---

### Requirement: Un `technical_context:` Inválido Falla Cerrado (REQ-30.30)

Cuando el planning home es determinable, si `axiom.yaml` no puede leerse como YAML o contiene un bloque `technical_context:` que no supera la validación (REQ-24.27), el sistema NO DEBE degradar a «sin contexto técnico». El status NO DEBE proyectar localizadores parciales del bloque afectado, DEBE señalar la causa y `axiom workspace validate` como continuación y DEBE marcar el archivado como bloqueado (`dependencies.archive` en `blocked`); las dependencias de las fases de planificación NO DEBEN pasar a `blocked` por esta causa. Un `axiom.yaml` legible que no declara `technical_context:` NO DEBE alterar el comportamiento por errores de validación ajenos a ese bloque. (Es la parte de contexto de REQ-24.18 de `spec-baseline-declaration`.)

#### Scenario: Un `technical_context:` inválido bloquea el archivado
- **DADO** un `axiom.yaml` con `technical_context.maintained.paths: []`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status no incluye `technicalContext`
- **Y** `dependencies.archive` es `blocked` con una causa que nombra la regla incumplida y `axiom workspace validate`
- **Y** las dependencias de las fases de planificación no son `blocked` por esa causa

#### Scenario: `axiom.yaml` ilegible sin puntero
- **DADO** un proyecto sin puntero cuyo `axiom.yaml` no es YAML válido
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status resuelve la raíz embebida con un aviso y `dependencies.archive` es `blocked` con la causa

#### Scenario: Errores ajenos al bloque no alteran el comportamiento
- **DADO** un `axiom.yaml` legible sin `technical_context:` y sin la sección `roles`
- **CUANDO** se ejecuta `axiom archive sync`
- **ENTONCES** el comando se comporta como antes de este cambio

---

### Requirement: Frontera de Configuración entre `axiom.yaml` y `openspec/config.yaml` (REQ-30.32)

La declaración estructural del contexto técnico (`technical_context`) DEBE residir en `axiom.yaml` y validarse como el resto de su esquema. Las reglas de conducta (`rules.<fase>`, `strict_tdd`, `artifact_store`) DEBEN permanecer en `openspec/config.yaml` y NO DEBEN trasladarse a `axiom.yaml`. Una clave `technical_context:` en `openspec/config.yaml` NO DEBE tener efecto. `governance.context` NO DEBE cambiar. `docs/openspec-config.md` DEBE documentar esta frontera. (Es REQ-24.21 de `spec-baseline-declaration`, sin la mención de `baseline`.)

#### Scenario: Una declaración en `openspec/config.yaml` no tiene efecto
- **DADO** un `openspec/config.yaml` con una clave `technical_context:` y un `axiom.yaml` sin ella
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status no contiene `technicalContext`

#### Scenario: `governance.context` convive con la declaración
- **DADO** un `axiom.yaml` con `governance.context` y `technical_context:`
- **CUANDO** se carga y se valida
- **ENTONCES** ambos se aceptan y `governance.context` conserva su valor sin interpretarse

#### Scenario: Frontera documentada
- **DADO** el texto de `docs/openspec-config.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que `technical_context` vive en `axiom.yaml` y que las reglas de conducta permanecen en `openspec/config.yaml`

