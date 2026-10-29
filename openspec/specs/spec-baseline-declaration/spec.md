# Especificación de Requerimientos: Declaración de la Baseline del Proyecto y Guarda contra una Segunda Baseline (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Rebanada:** B (declaración y guardas) · **Tipo:** capacidad nueva
> **Idioma:** Español (castellano peninsular)

## Propósito

Definir cómo declara un proyecto, en `axiom.yaml`, dónde está su *baseline* (la especificación viva que archive mantiene) y con qué formato; cómo se valida esa declaración, incluida su compatibilidad con el almacén de artefactos; cómo se proyecta en el status v2 sin romper su contrato; y cómo se garantiza que ningún creador de Axiom produce una segunda baseline cuando la declarada es del proyecto. Sin declaración, la baseline es OpenSpec (`<planning home>/specs/`) y todo el comportamiento es idéntico al actual.

La documentación del proyecto que solo se usa como entrada y nunca se modifica NO se declara aquí: se declara como contexto de referencia en la capacidad `project-technical-context`.

## Términos

| Término | Definición |
|---|---|
| Baseline | Conjunto de documentos que archive actualiza al integrar los cambios (la especificación viva). |
| Baseline por defecto | OpenSpec: `<planning home>/specs/<dominio>/spec.md`, con requisitos de encabezado `Requirement:`. |
| Baseline del proyecto | La declarada en `baseline.documents` de `axiom.yaml`, en la ubicación y el formato del proyecto. |
| Documento con perfil de requisito | Documento direccionable por requisito: archive lo integra por composición (capacidad `spec-baseline-integration`). |
| Documento de prosa | Documento sin perfil de requisito: archive lo integra por edición dirigida o solo lo registra. |
| Creador | Operación de Axiom que crea o actualiza `openspec/specs/` u `openspec/INDEX.md`. |
| Almacén de ficheros | Almacén de artefactos `openspec` o `hybrid` (`artifact_store` de `openspec/config.yaml`): el cambio vive como ficheros en su carpeta de `openspec/changes/`. Con `engram` o `none` no existe esa carpeta. |

## Ejemplo de declaración (no normativo)

```yaml
baseline:
  documents:
    - domain: requisitos-funcionales
      path: 01_Requisitos_Funcionales.md
      scenarios: omit
      requirement:
        heading: '^### (?P<id>RF-\d+): (?P<title>.+)$'
        level: 3
        identity: id
        render: '### {id}: {title}'
    - domain: modelo-de-datos
      path: 03_Modelo_Datos.md
      integration: directed
    - domain: flujos-de-negocio
      path: 04_Flujos_Negocio.md
      integration: register
```

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| REQ-24.13 | Bloque `baseline:` opcional y defecto OpenSpec | Sin bloque, comportamiento actual |
| REQ-24.14 | Documentos por dominio | Un dominio, un documento; rutas contenidas |
| REQ-24.15 | Perfil de requisito | Patrón con `id` y `title`, nivel, identidad y plantilla |
| REQ-24.16 | Documentos de prosa y modo de integración | `directed` o `register`, nunca ambos con perfil |
| REQ-24.17 | Validación de la declaración | Loader y `axiom workspace validate` (también el almacén, REQ-24.62), en español, código 1 |
| REQ-24.18 | Una declaración inválida falla cerrada | Nunca degrada a la baseline por defecto; con un puntero ilegible rechaza antes (REQ-24.7) |
| REQ-24.19 | Proyección de `baseline` en el status v2 | Localizadores aditivos, sin parsear YAML |
| REQ-24.20 | Contrato del status v2 solo aditivo | Ninguna clave existente cambia |
| REQ-24.21 | Frontera de configuración | Estructura en `axiom.yaml`, conducta en `openspec/config.yaml` |
| REQ-24.22 | Consulta única de baseline del proyecto | Todos los creadores responden igual |
| REQ-24.23 | Guarda en CLI, dashboard y TUI | Ninguna escritura crea `openspec/specs/` ni `INDEX.md` |
| REQ-24.24 | Guarda en las skills | `sdd-init` y la rama sin baseline de `sdd-archive` |
| REQ-24.25 | Lecturas de catálogo | Informan de que la baseline es externa |
| REQ-24.60 | Clave `scenarios` por documento | `include` por defecto (exige `level` ≤ 3) u `omit`; solo con perfil de requisito |
| REQ-24.62 | Almacén de ficheros para baseline del proyecto y contexto mantenido | `engram` y `none` se reportan y detienen el archivado |

## No cubre esta capacidad

- La migración de la baseline de un proyecto existente.
- Indexar la baseline del proyecto en livingdoc, dashboard o knowledge: solo se informa de que es externa (REQ-24.25).
- Las correcciones del arranque en frío de knowledge (nivel `####` del grabador frente a `###` del compose, dominios falsos del barrido, sobrescritura de `coldstart`), salvo las guardas de esta capacidad.

---

## 1. Capacidad: `spec-baseline-declaration`

El bloque `baseline:` de `axiom.yaml` declara los documentos de la especificación viva del proyecto. Su ausencia significa baseline OpenSpec. Con la declaración presente, Axiom lee y escribe esos documentos y ninguna operación crea una segunda baseline.

### Requirement: Bloque `baseline:` Opcional y Defecto OpenSpec (REQ-24.13)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `baseline:` con la lista `documents`. Sin el bloque, la baseline es la de OpenSpec por defecto y la lectura, la integración y el archivado DEBEN comportarse exactamente como antes de esta capacidad. Con el bloque, la baseline es la del proyecto: los documentos declarados, en su ubicación y formato, y Axiom NO DEBE consultar, crear ni actualizar `<planning home>/specs/` como baseline. El bloque se admite en las dos topologías (`monorepo-embedded` y `multirepo`). Un bloque `baseline:` sin documentos (lista ausente o vacía) NO es válido.
(Previously: el bloque se admitía en las tres topologías, incluida la retirada `monorepo-decoupled`)

#### Scenario: Sin bloque la baseline es OpenSpec
- **DADO** un `axiom.yaml` sin `baseline:`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status no contiene la clave `baseline`
- **Y** archive integra los deltas en `<planning home>/specs/<dominio>/spec.md` como antes

#### Scenario: Con bloque la baseline es la del proyecto
- **DADO** un `axiom.yaml` con `baseline.documents` que declara el dominio `requisitos-funcionales`
- **CUANDO** archive integra un cambio con delta para ese dominio
- **ENTONCES** la integración se aplica al documento declarado
- **Y** no se consulta ni se crea `<planning home>/specs/`

#### Scenario: El bloque se admite en las dos topologías
- **DADO** un `axiom.yaml` con `baseline:` válido en `monorepo-embedded` y en `multirepo`
- **CUANDO** se ejecuta `axiom workspace validate` en cada uno
- **ENTONCES** ninguno reporta error por la presencia del bloque

#### Scenario: Bloque sin documentos
- **DADO** un `axiom.yaml` con `baseline: {}` o con `documents: []`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español que la baseline declarada requiere al menos un documento
- **Y** el código de salida es `1`
### Requirement: Documentos por Dominio y Mapeo Dominio a Documento (REQ-24.14)

Cada entrada de `baseline.documents` DEBE declarar `domain` y `path`. `domain` DEBE ser un identificador en minúsculas, dígitos y guiones (`^[a-z0-9]+(-[a-z0-9]+)*$`) y es el mismo nombre del directorio `specs/<dominio>/` de las delta specs de un cambio. `path` DEBE ser una ruta relativa a `workspaceRoot` (separadores `/` o `\`) que apunta a un fichero regular existente y permanece dentro de `workspaceRoot`. Cada dominio DEBE mapear a un único documento y cada documento a un único dominio. Una delta spec solo puede integrarse en un dominio declarado (REQ-24.48).

#### Scenario: Declaración válida de dos dominios
- **DADO** un `axiom.yaml` con los dominios `requisitos-funcionales` y `modelo-de-datos` y los ficheros existentes en la raíz del repositorio de especificaciones
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** la validación no reporta error sobre `baseline`

#### Scenario: Dominio duplicado
- **DADO** dos entradas con `domain: requisitos-funcionales`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta un error que nombra el dominio duplicado

#### Scenario: Documento declarado en dos dominios
- **DADO** dos entradas con distinto `domain` y el mismo `path`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta un error que nombra la ruta y los dos dominios

#### Scenario: Ruta absoluta o que sale del repositorio de especificaciones
- **DADO** una entrada con `path: ../fuera.md` y otra con una ruta absoluta
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta un error por cada una nombrando la ruta y la regla de contención
- **Y** el código de salida es `1`

#### Scenario: Identificador de dominio no válido
- **DADO** una entrada con `domain: Requisitos Funcionales`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta un error que nombra el dominio y el formato admitido

---

### Requirement: Perfil de Requisito (REQ-24.15)

Una entrada de `baseline.documents` PUEDE incluir `requirement`, que hace el documento direccionable por requisito. DEBE declarar `heading` (expresión regular RE2 con los grupos con nombre `id` y `title`, que se aplica a cada línea de encabezado ATX), `level` (entero de 1 a 6: nivel de los encabezados de requisito) y `render` (plantilla de encabezado con los marcadores `{id}` y `{title}`, que comienza con exactamente `level` signos `#` seguidos de un espacio). PUEDE declarar `identity` con el valor `id` o `text`; si se omite, es `id`. Una línea es un encabezado de requisito solo si cumple `heading` y su nivel coincide con `level`. Las líneas dentro de bloques de código delimitados con vallas (tres acentos graves o tres virgulillas) NO se consideran encabezados.

#### Scenario: Perfil válido
- **DADO** el perfil `heading: '^### (?P<id>RF-\d+): (?P<title>.+)$'`, `level: 3`, `identity: id` y `render: '### {id}: {title}'`
- **CUANDO** se carga `axiom.yaml`
- **ENTONCES** la carga tiene éxito y el dominio queda declarado con perfil de requisito

#### Scenario: Patrón sin los grupos `id` y `title`
- **DADO** un perfil cuyo `heading` es `'^### (RF-\d+): (.+)$'` (grupos sin nombre)
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español que el patrón debe definir los grupos `id` y `title`, nombrando el dominio
- **Y** el código de salida es `1`

#### Scenario: Identidad desconocida
- **DADO** un perfil con `identity: nombre`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta que la identidad debe ser `id` o `text`, nombrando el dominio
- **Y** el código de salida es `1`

#### Scenario: Plantilla incoherente con el nivel
- **DADO** un perfil con `level: 3` y `render: '## {id}: {title}'`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta que `render` debe comenzar con tres signos `#`

#### Scenario: Patrón que no compila
- **DADO** un perfil con `heading: '^### (?P<id>RF-\d+: (?P<title>.+)$'` (paréntesis sin cerrar)
- **CUANDO** se carga `axiom.yaml`
- **ENTONCES** la carga falla con un mensaje en español que nombra el dominio y el error del patrón

---

### Requirement: Documentos de Prosa y Modo de Integración (REQ-24.16)

Una entrada sin `requirement` es un documento de prosa. PUEDE declarar `integration` con el valor `directed` (por defecto: archive integra mediante una edición dirigida con verificación nativa, según REQ-24.50 y REQ-24.51) o `register` (archive solo registra la integración pendiente, según REQ-24.53). `integration` NO DEBE declararse en una entrada con `requirement`, cuya integración es la composición.

#### Scenario: Documento de prosa con el valor por defecto
- **DADO** una entrada `modelo-de-datos` sin `requirement` ni `integration`
- **CUANDO** se carga la declaración
- **ENTONCES** su modo de integración efectivo es `directed`

#### Scenario: Documento de prosa en modo solo registro
- **DADO** una entrada con `integration: register`
- **CUANDO** se valida la declaración
- **ENTONCES** la declaración es válida

#### Scenario: Valor de integración desconocido
- **DADO** una entrada con `integration: auto`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta que `integration` debe ser `directed` o `register`, nombrando el dominio

#### Scenario: Integración declarada en un documento con perfil
- **DADO** una entrada con `requirement` e `integration: directed`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta que `integration` no se admite en un documento con perfil de requisito

---

### Requirement: Validación de la Declaración por el Loader y por `axiom workspace validate` (REQ-24.17)

Las reglas sintácticas de REQ-24.13 a REQ-24.16 y REQ-24.60, incluidas las claves desconocidas dentro de `baseline` y de sus entradas, DEBEN validarse al cargar `axiom.yaml`. Las reglas que dependen del sistema de ficheros (existencia de cada `path` como fichero regular y contención en `workspaceRoot`) y la compatibilidad de la declaración con el almacén de artefactos (REQ-24.62, que depende de `openspec/config.yaml`) DEBEN validarse en `axiom workspace validate`. Toda violación DEBE reportarse en español, nombrando el dominio, la ruta o la declaración afectados y la regla incumplida, y `axiom workspace validate` DEBE finalizar con código de salida `1` si hay alguna. Una declaración válida y compatible con el almacén DEBE aceptarse y el comando finaliza con código `0`.

#### Scenario: Declaración válida
- **DADO** un `axiom.yaml` con una baseline declarada, documentos existentes y un almacén de artefactos compatible (REQ-24.62)
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es `COMPLIANT` y el código de salida es `0`

#### Scenario: Documento inexistente
- **DADO** una entrada cuyo `path` no existe en el repositorio de especificaciones
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español la ruta inexistente y el dominio
- **Y** el código de salida es `1`

#### Scenario: Clave desconocida dentro del bloque
- **DADO** un bloque `baseline:` con una clave mal escrita (`documentss:`) o una entrada con `requierment:`
- **CUANDO** se carga `axiom.yaml`
- **ENTONCES** la carga falla nombrando la clave desconocida
- **Y** `axiom workspace validate` finaliza con código `1`

#### Scenario: Claves desconocidas fuera de los bloques nuevos
- **DADO** un `axiom.yaml` con una clave de primer nivel no reconocida y sin bloques nuevos
- **CUANDO** se carga
- **ENTONCES** la clave se ignora como hasta ahora y la carga tiene éxito

---

### Requirement: Una Declaración Inválida Falla Cerrada (REQ-24.18)

Cuando el planning home es determinable (sin puntero, o con un puntero cuyo destino carga la configuración base), si `axiom.yaml` no puede leerse como YAML, o contiene un bloque `baseline:` o `technical_context:` que no supera la validación, el sistema NO DEBE degradar a la baseline OpenSpec por defecto. El status NO DEBE proyectar localizadores parciales ni por defecto del bloque afectado, DEBE señalar la causa y `axiom workspace validate` como continuación y DEBE marcar el archivado como bloqueado (`dependencies.archive` en `blocked`); las dependencias de las fases de planificación NO DEBEN pasar a `blocked` por esta causa. Cuando la causa afecta a la baseline (fichero ilegible o bloque `baseline:` inválido), la consulta de REQ-24.22 responde `indeterminada` y los creadores de REQ-24.23 y REQ-24.24 DEBEN rechazar su escritura. Un `axiom.yaml` legible que no declara `baseline:` NO DEBE alterar el comportamiento actual de la baseline aunque tenga errores de validación ajenos a ese bloque. Con un puntero cuyo destino no puede leerse como YAML o no supera la validación de la configuración base, el planning home es desconocido y la resolución se rechaza antes (REQ-24.7, caso 2): no hay status que reportar ni consulta que responder, y los creadores rechazan por esa misma causa.

#### Scenario: Una declaración inválida bloquea el archivado
- **DADO** un `axiom.yaml` con `baseline.documents[0].requirement.identity: nombre`
- **CUANDO** se ejecuta `axiom sdd status --json --instructions`
- **ENTONCES** el status no incluye `baseline`
- **Y** `dependencies.archive` es `blocked` con una causa que nombra la identidad inválida y `axiom workspace validate`
- **Y** las dependencias de las fases de planificación no son `blocked` por esa causa

#### Scenario: Los creadores rechazan escribir
- **DADO** la misma declaración inválida
- **CUANDO** se ejecuta `axiom archive coldstart --change inc-99-ejemplo`
- **ENTONCES** se rechaza nombrando `axiom workspace validate` y no se crea `openspec/specs/`

#### Scenario: `axiom.yaml` con error de sintaxis YAML
- **DADO** un proyecto sin puntero cuyo `axiom.yaml` no es YAML válido
- **CUANDO** se ejecuta `axiom archive sync`
- **ENTONCES** se rechaza la escritura de `INDEX.md` nombrando la causa
- **Y** `axiom sdd status --json` sigue resolviendo la raíz embebida con un aviso (REQ-24.7) y marca el archivado como bloqueado

#### Scenario: Un `axiom.yaml` ilegible se rechaza con puntero y deja el status operativo sin él
- **DADO** el mismo `axiom.yaml` que no es YAML válido, apuntado por un `.axiom-workspace` en un caso y presente como `axiom.yaml` directo, sin puntero, de un proyecto con `openspec/` en el otro
- **CUANDO** se ejecuta `axiom sdd status --json` en cada caso
- **ENTONCES** con puntero se rechaza nombrando la causa y `axiom workspace validate`, porque el planning home es desconocido (REQ-24.7), y no se devuelve ningún status
- **Y** sin puntero el status resuelve la raíz embebida con un aviso, `dependencies.archive` es `blocked` con la causa y las dependencias de las fases de planificación no son `blocked`

#### Scenario: Un `technical_context:` inválido bloquea el archivado sin afectar a los creadores de baseline
- **DADO** un `axiom.yaml` sin `baseline:` y con `technical_context.maintained.paths: []`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status no incluye `technicalContext` y `dependencies.archive` es `blocked` con la causa y `axiom workspace validate`
- **Y** `axiom archive sync` se comporta como con la baseline por defecto

#### Scenario: Errores de validación ajenos a los bloques nuevos no alteran la baseline
- **DADO** un `axiom.yaml` legible sin `baseline:` y sin la sección `roles`
- **CUANDO** se ejecuta `axiom archive sync`
- **ENTONCES** la baseline sigue siendo la de OpenSpec por defecto y el comando se comporta como antes de esta capacidad

---

### Requirement: Proyección de `baseline` en el Status v2 (REQ-24.19)

Cuando `axiom.yaml` declara `baseline:` y la declaración es válida, el status v2 DEBE incluir el campo opcional `baseline` con la lista `documents` y, por cada documento, `domain`, `path` (ruta absoluta) e `integration` (`compose` para documentos con perfil de requisito; `directed` o `register` para documentos de prosa). El campo se proyecta con o sin cambio seleccionado. Cuando no hay declaración, el campo NO DEBE aparecer. Las skills DEBEN leer el localizador del status y NO DEBEN interpretar `axiom.yaml` por su cuenta.

#### Scenario: Status con baseline declarada
- **DADO** una baseline con tres documentos (uno con perfil, uno `directed` y uno `register`)
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `baseline.documents` contiene tres entradas con `domain`, `path` absoluto e `integration` igual a `compose`, `directed` y `register` respectivamente

#### Scenario: Status sin declaración
- **DADO** un `axiom.yaml` sin `baseline:`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el JSON no contiene la clave `baseline`

#### Scenario: Proyección sin cambio seleccionado
- **DADO** una baseline declarada y ningún cambio activo
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `changeName` es nulo y `baseline` está presente

#### Scenario: Contrato compartido documentado
- **DADO** el texto de `_shared/sdd-status-contract.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** documenta `baseline` como campo opcional y la regla de leer el localizador del status en lugar de `axiom.yaml`

---

### Requirement: Contrato del Status v2 Solo Aditivo (REQ-24.20)

La incorporación de `baseline` y `technicalContext` NO DEBE modificar el nombre, el tipo ni la semántica de ningún campo existente del status v2 y NO DEBE añadir claves obligatorias. Los campos nuevos DEBEN omitirse cuando no existe la declaración correspondiente. `planningHome.path` DEBE seguir siendo la ruta absoluta del directorio `openspec` y `planningHome.mode` DEBE seguir siendo `repo-local`. La prueba de claves exactas del status v2 (`status_v2_clean_break_test`) DEBE seguir pasando sin modificarse.

#### Scenario: Conjunto de claves obligatorias intacto
- **DADO** un workspace sin `baseline:` ni `technical_context:`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el conjunto de claves de primer nivel es exactamente el existente antes de este incremento

#### Scenario: Con declaración solo aparecen claves nuevas opcionales
- **DADO** un workspace con `baseline:` y `technical_context:`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** las únicas claves adicionales respecto al caso anterior son `baseline` y `technicalContext`

#### Scenario: `planningHome` no cambia con la declaración
- **DADO** un workspace con `baseline:` declarada
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `planningHome.path` es la ruta absoluta del directorio `openspec` y `planningHome.mode` es `repo-local`

---

### Requirement: Frontera de Configuración entre `axiom.yaml` y `openspec/config.yaml` (REQ-24.21)

La declaración estructural (`baseline` y `technical_context`) DEBE residir en `axiom.yaml` y validarse como el resto de su esquema. Las reglas de conducta (`rules.<fase>`, `strict_tdd`, `artifact_store`) DEBEN permanecer en `openspec/config.yaml` y NO DEBEN trasladarse a `axiom.yaml`. `governance.context` NO DEBE cambiar. `docs/openspec-config.md` DEBE documentar esta frontera.

#### Scenario: Una declaración en `openspec/config.yaml` no tiene efecto
- **DADO** un `openspec/config.yaml` con una clave `baseline:` y un `axiom.yaml` sin ella
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status no contiene `baseline`

#### Scenario: `governance.context` convive con la declaración
- **DADO** un `axiom.yaml` con `governance.context` y `technical_context:`
- **CUANDO** se carga y se valida
- **ENTONCES** ambos se aceptan y `governance.context` conserva su valor sin interpretarse

#### Scenario: Frontera documentada
- **DADO** el texto de `docs/openspec-config.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que `baseline` y `technical_context` viven en `axiom.yaml` y que las reglas de conducta permanecen en `openspec/config.yaml`

---

### Requirement: Consulta Única de Baseline del Proyecto (REQ-24.22)

El sistema DEBE ofrecer una única consulta «¿la baseline declarada es del proyecto?» con tres respuestas: `del proyecto` (`axiom.yaml` es legible y contiene un bloque `baseline:` válido), `por defecto` (no existe `axiom.yaml`, o es legible y no contiene `baseline:`, aunque tenga errores de validación ajenos a ese bloque) e `indeterminada` (`axiom.yaml` no puede leerse como YAML o su bloque `baseline:` es inválido). Todos los creadores de REQ-24.23 y REQ-24.24 DEBEN usarla y, para un mismo `axiom.yaml`, DEBEN coincidir en la respuesta. La consulta se resuelve con el primitivo de resolución de la capacidad `planning-home-resolution` (misma carpeta maestra y misma configuración). Con la respuesta `indeterminada`, los creadores DEBEN rechazar su escritura (REQ-24.18). Con un puntero cuyo destino no puede leerse como YAML o no supera la validación de la configuración base, la resolución se rechaza antes (REQ-24.7) y no hay respuesta: los creadores rechazan su escritura por esa misma causa.

#### Scenario: Todos los creadores coinciden
- **DADO** un `axiom.yaml` con baseline del proyecto
- **CUANDO** se evalúan `axiom init`, `axiom archive sync`, `axiom archive coldstart`, `axiom knowledge sweep`, `axiom knowledge crawl --record-unit`, `axiom knowledge crawl --finalize`, `axiom knowledge query --enrich` y la rama sin baseline de `sdd-archive`
- **ENTONCES** todos obtienen la respuesta `del proyecto`

#### Scenario: Sin `axiom.yaml` la respuesta es `por defecto`
- **DADO** un directorio con solo `openspec/`
- **CUANDO** se evalúa la consulta
- **ENTONCES** la respuesta es `por defecto` y los creadores operan como hasta ahora

#### Scenario: `axiom.yaml` ilegible o con baseline inválida
- **DADO** un proyecto sin puntero cuyo `axiom.yaml` tiene un error de sintaxis YAML, o contiene un bloque `baseline:` inválido
- **CUANDO** se evalúa la consulta
- **ENTONCES** la respuesta es `indeterminada`
- **Y** los creadores rechazan escribir `openspec/specs/` y `openspec/INDEX.md`

#### Scenario: Errores ajenos al bloque `baseline:`
- **DADO** un `axiom.yaml` legible sin `baseline:` y con un error de validación en la sección `roles`
- **CUANDO** se evalúa la consulta
- **ENTONCES** la respuesta es `por defecto`

---

### Requirement: Guarda en las Operaciones de la CLI, el Dashboard y la TUI (REQ-24.23)

Con la respuesta `del proyecto` (REQ-24.22), ninguna de las siguientes operaciones DEBE crear ni modificar `<planning home>/specs/` ni `<planning home>/INDEX.md`.

- Las operaciones cuyo único propósito es escribir esa ubicación DEBEN rechazarse sin escribir nada, con un mensaje que nombre la baseline declarada y una continuación, y las de la CLI DEBEN finalizar con código de salida distinto de cero: `axiom archive sync`, `axiom archive coldstart`, `axiom knowledge crawl --record-unit`, `axiom knowledge crawl --finalize`, y la sincronización del catálogo desde el dashboard web (`POST /api/archive/sync`) y desde la TUI.
- Las operaciones que además producen otro resultado útil DEBEN omitir esa escritura, informar de la omisión y completar el resto de su resultado con código de salida `0`: `axiom init` en cualquier perfil (que conserva el resto del andamiaje de su perfil: `openspec/changes/` siempre y `.axiom/inbox/skills/` solo en el perfil completo, el que no es `knowledge`), `axiom knowledge sweep` y `axiom knowledge query --enrich`.

#### Scenario: `axiom init` sobre una declaración existente
- **DADO** un directorio con un `axiom.yaml` que declara una baseline del proyecto
- **CUANDO** se ejecuta `axiom init`
- **ENTONCES** no existen `openspec/specs/` ni `openspec/INDEX.md`
- **Y** existen `openspec/changes/` y `.axiom/inbox/skills/`
- **Y** la salida informa de que la baseline es del proyecto y el código de salida es `0`

#### Scenario: `axiom init --profile=knowledge` con baseline del proyecto
- **DADO** el mismo `axiom.yaml`
- **CUANDO** se ejecuta `axiom init --profile=knowledge`
- **ENTONCES** no existen `openspec/specs/` ni `openspec/INDEX.md`
- **Y** existe `openspec/changes/` y no se crea `.axiom/inbox/skills/`

#### Scenario: `axiom archive coldstart` rechazado
- **DADO** una baseline del proyecto declarada
- **CUANDO** se ejecuta `axiom archive coldstart --change inc-99-ejemplo`
- **ENTONCES** el comando finaliza con código distinto de cero nombrando la baseline declarada y la continuación
- **Y** no existe `openspec/specs/`

#### Scenario: `axiom archive sync` rechazado
- **DADO** una baseline del proyecto declarada
- **CUANDO** se ejecuta `axiom archive sync`
- **ENTONCES** el comando finaliza con código distinto de cero y no se crea `openspec/INDEX.md`

#### Scenario: `axiom knowledge sweep` omite la siembra
- **DADO** una baseline del proyecto declarada
- **CUANDO** se ejecuta `axiom knowledge sweep --json`
- **ENTONCES** la salida contiene el análisis técnico y funcional y el código de salida es `0`
- **Y** `created_specs` está vacío y no existen `openspec/specs/` ni `openspec/INDEX.md`

#### Scenario: Registro y finalización de `axiom knowledge crawl` rechazados
- **DADO** una baseline del proyecto declarada
- **CUANDO** se ejecuta `axiom knowledge crawl --record-unit` o `axiom knowledge crawl --finalize`
- **ENTONCES** se rechaza sin escribir en `openspec/specs/` ni en `openspec/INDEX.md`

#### Scenario: `axiom knowledge query --enrich` no enriquece
- **DADO** una baseline del proyecto declarada
- **CUANDO** se ejecuta `axiom knowledge query "¿...?" --enrich`
- **ENTONCES** la consulta se responde, no se escribe `openspec/specs/` ni `openspec/INDEX.md` y la salida informa de la omisión

#### Scenario: Sincronización desde el dashboard web y la TUI
- **DADO** una baseline del proyecto declarada
- **CUANDO** se envía `POST /api/archive/sync` o se pulsa el atajo de sincronización de la TUI
- **ENTONCES** el servidor responde con un código de error de cliente (4xx) y la TUI muestra un mensaje de rechazo, ambos con la continuación
- **Y** no se escribe `INDEX.md`

#### Scenario: Sin baseline del proyecto el comportamiento previo se conserva
- **DADO** un `axiom.yaml` sin `baseline:`
- **CUANDO** se ejecutan las operaciones anteriores
- **ENTONCES** se comportan como antes de esta capacidad

---

### Requirement: Guarda en las Skills Creadoras de Baseline (REQ-24.24)

El texto de la skill `sdd-init` NO DEBE prescribir la creación de `openspec/specs/` cuando el status declara `baseline`. La rama de `sdd-archive` para un dominio sin especificación viva (promoción de un delta completo) NO DEBE copiar el delta a `openspec/specs/<dominio>/` con baseline del proyecto: un dominio no declarado se rechaza (REQ-24.48) y uno declarado se integra según REQ-24.47 o REQ-24.50. El texto de ambas skills DEBE prescribirlo.

#### Scenario: `sdd-init` no prescribe `openspec/specs/` con baseline del proyecto
- **DADO** el texto de la skill `sdd-init`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que, si el status proyecta `baseline`, no se crea `openspec/specs/`

#### Scenario: Rama sin baseline de `sdd-archive` con baseline del proyecto
- **DADO** una baseline del proyecto que no declara el dominio `nuevo-dominio` y un cambio con `specs/nuevo-dominio/spec.md`
- **CUANDO** se ejecuta el archivado
- **ENTONCES** se rechaza nombrando el dominio y la continuación
- **Y** no existe `openspec/specs/nuevo-dominio/`

#### Scenario: Sin baseline del proyecto la rama sin baseline se conserva
- **DADO** un `axiom.yaml` sin `baseline:` y un dominio sin especificación viva
- **CUANDO** se ejecuta el archivado
- **ENTONCES** el delta completo se copia mecánicamente a `openspec/specs/<dominio>/spec.md` como hasta ahora

---

### Requirement: Las Lecturas de Catálogo Informan de que la Baseline es Externa (REQ-24.25)

Con baseline del proyecto declarada, las lecturas de catálogo de especificaciones vivas (`axiom archive list`, `axiom archive show`, el catálogo del dashboard web, la pantalla de documentación viva de la TUI y la búsqueda spec-first de `axiom knowledge query`) NO DEBEN indexar los documentos de la baseline del proyecto y DEBEN informar de que la baseline es externa y de dónde se encuentra (dominios y rutas declarados), de modo que un catálogo vacío no se interprete como ausencia de especificación.

#### Scenario: `axiom archive list` con baseline externa
- **DADO** una baseline del proyecto con dos documentos declarados
- **CUANDO** se ejecuta `axiom archive list`
- **ENTONCES** la salida informa de que la baseline es externa y lista los dos dominios con sus rutas
- **Y** no indexa su contenido y el código de salida es `0`

#### Scenario: `axiom archive show` de un dominio declarado
- **DADO** una baseline del proyecto que declara `requisitos-funcionales`
- **CUANDO** se ejecuta `axiom archive show requisitos-funcionales`
- **ENTONCES** informa de que el dominio pertenece a la baseline externa y muestra su ruta

#### Scenario: Búsqueda spec-first de knowledge
- **DADO** una baseline del proyecto declarada
- **CUANDO** se ejecuta `axiom knowledge query "¿...?"`
- **ENTONCES** la salida informa de que la búsqueda spec-first no cubre la baseline externa

---

### Requirement: Clave `scenarios` por Documento con Perfil de Requisito (REQ-24.60)

Una entrada de `baseline.documents` con `requirement` PUEDE declarar `scenarios` con el valor `include` o `omit`; si se omite, es `include`. La clave determina si, al integrar un requisito en el documento, los escenarios de la delta spec pasan al bloque de requisito del proyecto (REQ-24.61): `include` integra descripción y escenarios; `omit` integra solo la descripción, de modo que el formato de escenarios de OpenSpec no se impone en la especificación viva del proyecto. `scenarios` NO DEBE declararse en una entrada sin `requirement` (documento de prosa) y cualquier otro valor DEBE rechazarse. Con `include` (declarado o por defecto), `requirement.level` DEBE ser menor o igual que 3: los encabezados de escenario de las delta specs son `#### Scenario:` (nivel 4) y, con un nivel de requisito igual o mayor que 4, cerrarían el bloque del requisito (REQ-24.41) y dejarían parte del cuerpo fuera de él (REQ-24.42). La validación DEBE reportar esa incompatibilidad al cargar `axiom.yaml`, en español, nombrando el dominio, el `level` declarado y las dos salidas: declarar `scenarios: omit` o un `level` menor o igual que 3. La baseline OpenSpec por defecto no declara esta clave y siempre incluye los escenarios.

#### Scenario: Valor por defecto
- **DADO** una entrada con `requirement` y sin `scenarios`
- **CUANDO** se carga la declaración
- **ENTONCES** su valor efectivo es `include`

#### Scenario: Valor `omit` explícito
- **DADO** una entrada con `requirement` y `scenarios: omit`
- **CUANDO** se carga y se valida la declaración
- **ENTONCES** la declaración es válida y su valor efectivo es `omit`

#### Scenario: Valor no válido
- **DADO** una entrada con `requirement` y `scenarios: ninguno`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español que `scenarios` debe ser `include` u `omit`, nombrando el dominio
- **Y** el código de salida es `1`

#### Scenario: Clave en un documento de prosa
- **DADO** una entrada sin `requirement` con `scenarios: omit`
- **CUANDO** se valida la declaración
- **ENTONCES** se reporta que `scenarios` solo se admite en documentos con perfil de requisito, nombrando el dominio
- **Y** el código de salida es `1`

#### Scenario: `include` por defecto con un nivel de requisito superior a 3
- **DADO** una entrada con `requirement.level: 4` y sin `scenarios` (valor efectivo `include`)
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español que los encabezados `#### Scenario:` cerrarían el bloque de nivel 4, nombrando el dominio y el `level`
- **Y** el mensaje propone declarar `scenarios: omit` o un `level` menor o igual que 3
- **Y** el código de salida es `1`

#### Scenario: `include` explícito con un nivel de requisito superior a 3
- **DADO** una entrada con `requirement.level: 5` y `scenarios: include`
- **CUANDO** se carga `axiom.yaml`
- **ENTONCES** la carga falla con la misma incompatibilidad, nombrando el dominio y las dos salidas

#### Scenario: `omit` admite un nivel de requisito superior a 3
- **DADO** una entrada con `requirement.level: 4` y `scenarios: omit`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** la declaración es válida y no se reporta error por el nivel

#### Scenario: El nivel 3 con `include` es válido
- **DADO** una entrada con `requirement.level: 3` y sin `scenarios`
- **CUANDO** se valida la declaración
- **ENTONCES** la declaración es válida

---

### Requirement: Baseline del Proyecto y Contexto Mantenido Exigen un Almacén de Artefactos de Ficheros (REQ-24.62)

Una baseline del proyecto (`baseline:` con documentos) y un contexto técnico mantenido (`technical_context.maintained`) DEBEN usarse con un almacén de ficheros (`openspec` o `hybrid`), porque la integración de archive lee y escribe ficheros de la carpeta del cambio: el delta de contexto (REQ-24.31) y la traza de integración (REQ-24.52). El almacén es el que declara `artifact_store` en `openspec/config.yaml` (REQ-24.21) y, si no se declara, el que resuelve el status para el cambio. Con `engram` o `none`:

- `axiom workspace validate` DEBE reportar un error en español que nombre la declaración afectada (`baseline` o `technical_context.maintained`), el almacén declarado y la continuación (declarar `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`, o retirar la declaración), y finalizar con código de salida `1`. La validación solo evalúa el valor declarado: sin `artifact_store` declarado no reporta nada por este motivo.
- El status DEBE marcar el archivado como bloqueado (`dependencies.archive` en `blocked`) con esa causa y la continuación. Las dependencias de las fases de planificación y de `apply` NO DEBEN pasar a `blocked` por ella y el status sigue proyectando `baseline` y `technicalContext`: la declaración no es inválida a efectos de REQ-24.18.
- Archive DEBE detenerse (REQ-24.56, caso g).

Una declaración solo de contexto de referencia (`technical_context.reference`) y la ausencia de baseline del proyecto y de contexto mantenido NO exigen un almacén de ficheros: un proyecto con almacén `engram` o `none` sin esas declaraciones conserva el comportamiento actual.

#### Scenario: Baseline del proyecto con almacén `engram`
- **DADO** un `axiom.yaml` con `baseline:` válido y un `openspec/config.yaml` con `artifact_store: engram`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español que la baseline del proyecto exige un almacén de ficheros, nombrando `baseline`, `engram` y la continuación
- **Y** el código de salida es `1`

#### Scenario: Contexto mantenido con almacén `none`
- **DADO** un `axiom.yaml` con `technical_context.maintained` válido y un `openspec/config.yaml` con `artifact_store: none`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta el mismo error nombrando `technical_context.maintained` y `none`
- **Y** el código de salida es `1`

#### Scenario: Los almacenes de ficheros y la ausencia de declaración no se reportan
- **DADO** la misma declaración con `artifact_store: openspec`, con `artifact_store: hybrid` y sin `artifact_store` declarado
- **CUANDO** se ejecuta `axiom workspace validate` en cada caso
- **ENTONCES** ninguno reporta error por el almacén

#### Scenario: El status bloquea el archivado sin bloquear la planificación
- **DADO** una baseline del proyecto, `artifact_store: engram` declarado y un cambio resuelto en ese almacén
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `dependencies.archive` es `blocked` con una causa que nombra el almacén `engram` y la continuación
- **Y** `baseline` sigue proyectada y las dependencias de las fases de planificación y de `apply` no son `blocked` por esa causa

#### Scenario: El almacén declarado prevalece sobre el hallado en disco
- **DADO** una baseline del proyecto, `artifact_store: none` declarado y un cambio presente en `openspec/changes/`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** `dependencies.archive` es `blocked` con una causa que nombra el almacén `none`

#### Scenario: Sin baseline del proyecto ni contexto mantenido el almacén no se exige
- **DADO** un `axiom.yaml` sin `baseline:` y con solo `technical_context.reference` (o sin bloques nuevos), y `artifact_store: engram`
- **CUANDO** se ejecutan `axiom workspace validate` y `axiom sdd status --json`
- **ENTONCES** no se reporta error por el almacén y `dependencies.archive` no se bloquea por esa causa
