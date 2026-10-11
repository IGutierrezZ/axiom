# Especificación Viva: Motor de Documentación Viva y Adopción Orgánica en Archive (INC-07)

## Propósito

Definir de forma ejecutable, determinista y verificable los requerimientos funcionales, no funcionales y escenarios BDD para transformar la fase de cierre (`Archive`) en un mantenedor continuo de especificaciones vivas y en un motor de adopción orgánica (*Zero-Doc Cold Start*) en Axiom: indexador canónico de especificaciones vivas (`openspec/specs/`), generador determinista del catálogo maestro `openspec/INDEX.md`, sintetizador orgánico para proyectos legados, subcomandos CLI `axiom archive` y la extensión visual del Dashboard Web local (`axiom ui`).

---

## 1. Capacidad: `livingdoc-spec-indexer`

El paquete `internal/livingdoc` provee capacidades de inspección, parseo y catalogación continua de todas las especificaciones vivas consolidadas en el workspace (`openspec/specs/`).

### Requirement: Escaneo y Extracción de Especificaciones Vivas (REQ-1.1)

El indexador DEBE recorrer recursivamente el directorio `<planning home>/specs/` (en la baseline por defecto, `openspec/specs/`; el planning home lo resuelve el primitivo de la capacidad `planning-home-resolution`, REQ-24.1 y REQ-24.10), localizando todos los archivos `spec.md`. Para cada archivo DEBE extraer:
- Identificador de dominio (nombre de la subcarpeta).
- Título principal de la especificación (encabezado `# `).
- Propósito o descripción introductoria.
- Lista completa de requerimientos canónicos (`### Requirement: ...`).
- Lista de escenarios BDD vinculados (`#### Scenario: ...`).

Cuando en un proyecto `monorepo-embedded` con `specs_repository` distinto de `""`, `.` y `openspec` existe `<raíz>/<specs_repository>/specs` y difiere de `<planning home>/specs`, el indexador DEBE leer solo `<planning home>/specs` y emitir un aviso que nombre ambas rutas. Con baseline del proyecto declarada, el indexador NO DEBE indexar sus documentos y DEBE informar de que la baseline es externa (REQ-24.25).
(Previously: recorría `openspec/specs/` o `<specs_repository>/specs` según el valor de `specs_repository`)

#### Scenario: Indexación exitosa de especificaciones vivas existentes
- **DADO** un directorio `openspec/specs/` con especificaciones vivas (`workspace-topology`, `structured-handoffs`, `multi-role-fan-out`, etc.)
- **CUANDO** el indexador procesa el repositorio de especificaciones
- **ENTONCES** genera una estructura `LivingCatalog` con una entrada `LivingSpecEntry` por cada dominio
- **Y** cada entrada reporta su recuento exacto de requerimientos y escenarios BDD

#### Scenario: Directorio de especificaciones vacío o inexistente
- **DADO** un proyecto que parte de cero sin la carpeta `openspec/specs/`
- **CUANDO** se invoca el indexador
- **ENTONCES** retorna un catálogo vacío con 0 especificaciones sin generar errores fatales

#### Scenario: Raíz resuelta en el workspace canónico
- **DADO** el workspace canónico con `<maestra>/repo-specs/openspec/specs/workspace-topology/spec.md`
- **CUANDO** el indexador procesa el workspace desde `<maestra>`
- **ENTONCES** el catálogo incluye el dominio `workspace-topology` leído desde el planning home resuelto

#### Scenario: Ubicación antigua divergente emite un aviso
- **DADO** un proyecto `monorepo-embedded` con `specs_repository: specs`, un `specs/specs/antiguo/spec.md` y un `openspec/specs/vigente/spec.md`
- **CUANDO** el indexador procesa el proyecto
- **ENTONCES** el catálogo incluye `vigente` y no incluye `antiguo`
- **Y** se emite un aviso que nombra `specs/specs` y `openspec/specs`

#### Scenario: Con baseline del proyecto se informa y no se indexa
- **DADO** un `axiom.yaml` que declara una baseline del proyecto con dos documentos
- **CUANDO** el indexador procesa el workspace
- **ENTONCES** el catálogo no contiene entradas de esos documentos
- **Y** el resultado informa de que la baseline es externa y de los dominios y rutas declarados

---

### Requirement: Generación Determinista de `openspec/INDEX.md` (REQ-1.2)

El motor DEBE ser capaz de compilar y escribir un documento canónico `INDEX.md` en el planning home (`<planning home>/INDEX.md`, que en la baseline por defecto es `openspec/INDEX.md`) que resuma el estado consolidado de la arquitectura y contratos vivos del sistema. El índice DEBE incluir:
1. Encabezado maestro y fecha/hora de la última sincronización.
2. Tabla resumen de especificaciones vivas con: Dominio, Título, Requerimientos, Escenarios y Enlace relativo al archivo `spec.md`.
3. Desglose por dominios con el listado de requerimientos canónicos y sus escenarios.

Con baseline del proyecto declarada, el motor NO DEBE crear ni actualizar `INDEX.md` (REQ-24.23).
(Previously: escribía siempre `openspec/INDEX.md`, también cuando la baseline era del proyecto)

#### Scenario: Generación y persistencia de `openspec/INDEX.md`
- **DADO** un catálogo de especificaciones vivas indexado
- **CUANDO** el servicio ejecuta el método de sincronización de índice
- **ENTONCES** escribe el archivo `INDEX.md` en el planning home (`openspec/INDEX.md` en la baseline por defecto) en el sistema de archivos
- **Y** todos los enlaces a las especificaciones son relativos y navegables

#### Scenario: Con baseline del proyecto no se genera el índice
- **DADO** un `axiom.yaml` que declara una baseline del proyecto
- **CUANDO** el servicio ejecuta el método de sincronización de índice
- **ENTONCES** la operación se rechaza sin escribir `INDEX.md`
- **Y** el mensaje nombra la baseline declarada y la continuación

---

### Requirement: Detección de Inconsistencias o Requerimientos Huérfanos (REQ-1.3)

Al indexar, el motor DEBE advertir si alguna especificación carece de título, contiene requerimientos sin escenarios BDD asociados o tiene identificadores de requerimiento duplicados.

#### Scenario: Advertencia ante requerimiento sin escenarios
- **DADO** un archivo `spec.md` con un encabezado `### Requirement: ...` que no contiene ningún `#### Scenario:`
- **CUANDO** se ejecuta la indexación
- **ENTONCES** el catálogo registra el requerimiento pero añade una advertencia en `SyncReport`

---

## 2. Capacidad: `livingdoc-organic-coldstart`

El motor de adopción orgánica permite introducir SDD en proyectos existentes sin documentación previa (*Zero-Doc Cold Start*).

### Requirement: Síntesis de Especificación Viva a partir de un Cambio Archivado (REQ-2.1)

Cuando un incremento concluye en un proyecto donde el dominio correspondiente aún no cuenta con especificación viva en `<planning home>/specs/<dominio>/` (en la baseline por defecto, `openspec/specs/<dominio>/`), el sintetizador DEBE tomar los artefactos del incremento (`spec.md`, `proposal.md`, `verify-report.md`) y generar de forma determinista la especificación viva inicial en `<planning home>/specs/<dominio>/spec.md`. Con baseline del proyecto declarada, el sintetizador NO DEBE crear `<planning home>/specs/<dominio>/` ni generar la especificación viva: DEBE rechazar la operación nombrando la baseline declarada y la continuación (REQ-24.23).
(Previously: sintetizaba siempre en `openspec/specs/<dominio>/`, ignorando `specs_repository` y la baseline del proyecto)

#### Scenario: Adopción orgánica de un nuevo dominio
- **DADO** un cambio archivado o en fase de cierre para el dominio `billing-service`
- **Y** que no existe previamente `openspec/specs/billing-service/spec.md`
- **CUANDO** se invoca la operación de *Cold Start*
- **ENTONCES** se crea la carpeta `openspec/specs/billing-service/`
- **Y** se genera el archivo `spec.md` inicial con los requerimientos y contratos validados en el cambio

#### Scenario: Arranque en frío rechazado con baseline del proyecto
- **DADO** un `axiom.yaml` que declara una baseline del proyecto y un cambio para el dominio `billing-service`
- **CUANDO** se invoca la operación de *Cold Start*
- **ENTONCES** se rechaza nombrando la baseline declarada y la continuación
- **Y** no existe `openspec/specs/billing-service/`

---

### Requirement: Preservación de Especificaciones Existentes (REQ-2.2)

El sintetizador de adopción orgánica NO DEBE sobreescribir una especificación viva previamente consolidada si ya existe en `openspec/specs/<dominio>/spec.md`, devolviendo un error controlado salvo que se solicite explícitamente su actualización.

#### Scenario: Protección contra sobreescritura de specs vivas existentes
- **DADO** una especificación viva que ya existe en `openspec/specs/workspace-topology/spec.md`
- **CUANDO** se intenta ejecutar *Cold Start* apuntando a dicho dominio
- **ENTONCES** la operación es rechazada con un mensaje de protección de especificación existente

---

## 3. Capacidad: `axiom-cli-archive`

La CLI de Axiom en `cmd/axiom/main.go` expone el grupo de comandos `axiom archive`.

### Requirement: Comando `axiom archive sync` (REQ-3.1)

El comando `axiom archive sync [--path <dir>]` (donde `--path` es el directorio de partida de la resolución del planning home) DEBE:
1. Re-escanear todas las especificaciones vivas en `<planning home>/specs/` (en la baseline por defecto, `openspec/specs/`).
2. Generar y sobrescribir de forma determinista el archivo `<planning home>/INDEX.md` (en la baseline por defecto, `openspec/INDEX.md`).
3. Imprimir por consola el resumen con el total de especificaciones, requerimientos y escenarios sincronizados.

Con baseline del proyecto declarada, el comando DEBE rechazar la operación sin escribir nada, nombrar la baseline declarada y la continuación, y finalizar con código de salida distinto de cero (REQ-24.23).
(Previously: reescaneaba `openspec/specs/` y reescribía `openspec/INDEX.md` siempre, también con baseline del proyecto)

#### Scenario: Sincronización exitosa desde CLI
- **DADO** un workspace de Axiom con especificaciones vivas
- **CUANDO** el usuario ejecuta `axiom archive sync`
- **ENTONCES** se actualiza `openspec/INDEX.md`
- **Y** la salida por consola muestra el informe de sincronización con código de salida 0

#### Scenario: Sincronización rechazada con baseline del proyecto
- **DADO** un `axiom.yaml` que declara una baseline del proyecto
- **CUANDO** el usuario ejecuta `axiom archive sync`
- **ENTONCES** el comando finaliza con código de salida distinto de cero nombrando la baseline declarada y la continuación
- **Y** no se crea ni se modifica `openspec/INDEX.md`

#### Scenario: Sincronización desde el workspace canónico
- **DADO** el workspace canónico con especificaciones vivas en `<maestra>/repo-specs/openspec/specs/`
- **CUANDO** el usuario ejecuta `axiom archive sync --path <maestra>`
- **ENTONCES** se actualiza `<maestra>/repo-specs/openspec/INDEX.md`

---

### Requirement: Comando `axiom archive list` (REQ-3.2)

El comando `axiom archive list [--path <dir>]` DEBE presentar una tabla formateada con las especificaciones vivas activas, sus títulos, cantidad de requerimientos y escenarios BDD.

#### Scenario: Listado de especificaciones en consola
- **DADO** un workspace con especificaciones vivas
- **CUANDO** el usuario ejecuta `axiom archive list`
- **ENTONCES** se imprime una tabla con las columnas DOMINIO, REQUISITOS, ESCENARIOS y TÍTULO

---

### Requirement: Comando `axiom archive show` (REQ-3.3)

El comando `axiom archive show <dominio> [--path <dir>]` DEBE mostrar en consola el detalle completo de la especificación viva para el dominio solicitado, incluyendo su propósito, lista de requerimientos y escenarios.

#### Scenario: Consulta de una especificación viva por dominio
- **DADO** una especificación viva existente para `workspace-topology`
- **CUANDO** el usuario ejecuta `axiom archive show workspace-topology`
- **ENTONCES** la CLI imprime el detalle estructurado de la especificación y sus requerimientos

---

### Requirement: Comando `axiom archive coldstart` (REQ-3.4)

El comando `axiom archive coldstart --change <nombre> [--domain <dominio>] [--path <dir>]` DEBE promover un cambio verificado a especificación viva cuando no existía previamente, en el planning home resuelto. Con baseline del proyecto declarada, el comando DEBE rechazar la operación sin escribir nada, nombrar la baseline declarada y la continuación, y finalizar con código de salida distinto de cero (REQ-24.23).
(Previously: promovía siempre a `openspec/specs/`, también con baseline del proyecto)

#### Scenario: Ejecución de coldstart para un cambio nuevo
- **DADO** un cambio completado con sus artefactos `spec.md` y `verify-report.md`
- **CUANDO** el usuario ejecuta `axiom archive coldstart --change 2026-09-14-inc-01-axiom-identity-workspace-topology`
- **ENTONCES** se consolida la especificación viva en `openspec/specs/`
- **Y** se ejecuta automáticamente la sincronización de `openspec/INDEX.md`

#### Scenario: Coldstart rechazado con baseline del proyecto
- **DADO** un `axiom.yaml` que declara una baseline del proyecto
- **CUANDO** el usuario ejecuta `axiom archive coldstart --change inc-99-ejemplo`
- **ENTONCES** el comando finaliza con código de salida distinto de cero nombrando la baseline declarada y la continuación
- **Y** no existe `openspec/specs/`

---

## 4. Capacidad: `axiom-dashboard-living-specs`

El Dashboard Web local (`internal/dashboard/`) expone la documentación viva a través de su API REST JSON y la renderiza en una nueva vista de la interfaz SPA.

### Requirement: Endpoints REST para Especificaciones Vivas (REQ-4.1)

El servidor web DEBE exponer las siguientes rutas REST:
- `GET /api/archive/specs`: Retorna el catálogo completo `LivingCatalog` serializado en JSON.
- `GET /api/archive/specs/{domain}`: Retorna el contenido estructurado y texto Markdown de la spec viva del dominio solicitado.
- `POST /api/archive/sync`: Ejecuta la re-indexación y sincronización de `openspec/INDEX.md` desde la Web UI.

Con baseline del proyecto declarada: `GET /api/archive/specs` DEBE responder 200 con la lista de especificaciones vacía, un indicador de baseline externa y los dominios y rutas declarados (los nombres de campo los fija el diseño); `GET /api/archive/specs/{domain}` de un dominio declarado DEBE responder con un error de cliente (4xx) que informa de que la baseline es externa y nombra su ruta; y `POST /api/archive/sync` DEBE responder con un error de cliente (4xx) con la continuación, sin escribir `INDEX.md` (REQ-24.23 y REQ-24.25).
(Previously: los tres endpoints operaban siempre sobre `openspec/specs/` y `openspec/INDEX.md`)

#### Scenario: Petición HTTP a `/api/archive/specs`
- **DADO** el servidor del Dashboard Web en ejecución
- **CUANDO** se efectúa una petición `GET` a `/api/archive/specs`
- **ENTONCES** responde con código HTTP 200 y cabecera `Content-Type: application/json`
- **Y** el cuerpo contiene la lista de especificaciones vivas con sus requerimientos

#### Scenario: Catálogo con baseline del proyecto
- **DADO** el servidor del Dashboard Web con un workspace cuya baseline es del proyecto
- **CUANDO** se efectúa una petición `GET` a `/api/archive/specs`
- **ENTONCES** responde con código HTTP 200, una lista de especificaciones vacía y un indicador de baseline externa con los dominios y rutas declarados

#### Scenario: Sincronización rechazada con baseline del proyecto
- **DADO** el mismo servidor
- **CUANDO** se efectúa una petición `POST` a `/api/archive/sync`
- **ENTONCES** responde con un código de error de cliente (4xx) cuyo cuerpo nombra la baseline declarada y la continuación
- **Y** no se escribe `INDEX.md`

---

### Requirement: Interfaz Web SPA con Explorador de Documentación Viva (REQ-4.2)

La interfaz SPA embebida (`assets/index.html`, `app.js`, `style.css`) DEBE incluir una nueva pestaña de navegación **"Especificaciones Vivas"** que permita:
1. Visualizar tarjetas de resumen con el total de especificaciones vivas, requerimientos y escenarios BDD consolidados en el sistema.
2. Disponer de un botón de acción rápida para re-sincronizar el catálogo (`↻ Sincronizar Catálogo`).
3. Navegar interactivamente por los dominios de especificación y leer su contenido formateado con visor de requerimientos y escenarios.

Con baseline del proyecto declarada, la pestaña DEBE mostrar un aviso de baseline externa con los dominios y rutas declarados en lugar del catálogo vacío, y el botón de sincronización DEBE mostrar el rechazo con su continuación sin escribir nada (REQ-24.25).
(Previously: la pestaña mostraba siempre el catálogo de `openspec/specs/`)

#### Scenario: Exploración de especificaciones vivas en el navegador
- **DADO** el usuario con el Dashboard abierto en la pestaña "Especificaciones Vivas"
- **CUANDO** selecciona una especificación de la lista
- **ENTONCES** se carga el detalle del dominio con su propósito, requerimientos y escenarios BDD
- **Y** se provee un botón para sincronizar el índice en un solo clic

#### Scenario: Pestaña con baseline del proyecto
- **DADO** el usuario con el Dashboard abierto en la pestaña "Especificaciones Vivas" sobre un workspace con baseline del proyecto
- **CUANDO** carga la pestaña
- **ENTONCES** se muestra el aviso de baseline externa con los dominios y rutas declarados
- **Y** al pulsar el botón de sincronización se muestra el rechazo con su continuación y no se escribe `INDEX.md`
