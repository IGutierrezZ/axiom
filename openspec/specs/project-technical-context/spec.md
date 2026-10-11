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

`axiom.yaml` PUEDE declarar un bloque de primer nivel `technical_context:` con los modos `maintained` (contexto técnico vivo que el flujo SDD mantiene al día) y `reference` (documentación que solo se lee y nunca se modifica). Cada modo declarado DEBE incluir `paths` (lista no vacía de ficheros o directorios) y PUEDE incluir `entry` (fichero que las fases lectoras leen por defecto, que DEBE estar contenido en `paths` del mismo modo). Las rutas son relativas a `workspaceRoot`, se admiten `/` y `\` como separadores, DEBEN permanecer dentro de la carpeta maestra y NO PUEDEN ser absolutas. Una ruta NO DEBE pertenecer a ambos modos ni a `baseline.documents`; dos rutas solapan cuando son iguales o una contiene a la otra. Sin el bloque no hay contexto técnico declarado.

#### Scenario: Declaración con los dos modos
- **DADO** un `axiom.yaml` con `maintained` (`entry: context/TECHNICAL_CONTEXT.md`, `paths: [context/]`) y `reference` (`paths: [00_Glosario.md]`)
- **CUANDO** se carga y se valida
- **ENTONCES** la declaración es válida

#### Scenario: Solo contexto de referencia
- **DADO** un `axiom.yaml` con `reference` y sin `maintained`
- **CUANDO** se carga y se valida
- **ENTONCES** la declaración es válida
- **Y** ningún documento queda sujeto a mantenimiento por archive
- **Y** la baseline sigue siendo la de OpenSpec por defecto, salvo que `axiom.yaml` declare `baseline:`

#### Scenario: Ruta en ambos modos
- **DADO** una ruta `context/a.md` en `maintained.paths` y la misma ruta en `reference.paths`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta un error que nombra la ruta y los dos modos

#### Scenario: Solapamiento por contención
- **DADO** `context/` en `maintained.paths` y `context/a.md` en `reference.paths`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta un error que nombra ambas rutas

#### Scenario: Ruta compartida con la baseline
- **DADO** `03_Modelo_Datos.md` en `baseline.documents` y en `reference.paths`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta un error que nombra la ruta, la baseline y el modo

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

La sintaxis (modos admitidos, claves desconocidas, `paths` no vacío, rutas relativas y contenidas, `entry` contenida en `paths`, solapes entre modos y con `baseline.documents`) DEBE validarse al cargar `axiom.yaml`. Una ruta declarada que no existe NO DEBE ser un error: DEBE reportarse como advertencia sin alterar el código de salida, porque un documento mantenido puede crearse en archive (REQ-24.55). Toda violación DEBE reportarse en español nombrando la ruta y la regla, y `axiom workspace validate` DEBE finalizar con código de salida `1` si hay alguna. Con `maintained` declarado, `axiom workspace validate` DEBE reportar además como error la incompatibilidad con un almacén de artefactos sin ficheros (`engram` o `none`), según REQ-24.62 de la capacidad `spec-baseline-declaration`.

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

Ninguna fase del flujo SDD (explore, propose, spec, design, tasks, apply, verify y archive) NI la integración de archive DEBE modificar, crear ni eliminar ficheros bajo las rutas del modo `reference`. El texto de cada skill de fase DEBE prescribirlo y la verificación nativa de la integración (REQ-24.51) DEBE rechazar cualquier diff que las afecte.

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

#### Scenario: Las skills lo prescriben
- **DADO** los textos de las skills de las fases del flujo SDD
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** cada uno prohíbe escribir en las rutas del modo `reference`

---

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
