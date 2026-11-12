# Delta para living-documentation

> Parte del estado vigente: el indexador, el índice, la síntesis, los comandos `axiom archive` y el dashboard NO cambian de conducta con el planning home por defecto. Esta delta solo retira los párrafos y escenarios de baseline externa (REQ-24.23 y REQ-24.25 de `spec-baseline-declaration`, que se retiran con la baseline ajena): catálogo vacío con indicador de baseline externa, rechazo de `sync`, `coldstart` y de `POST /api/archive/sync`, y aviso en la pestaña.
> Nota de coordinación: REQ-3.2 (`axiom archive list`) y REQ-3.3 (`axiom archive show`) no cambian; el grupo `axiom archive` gana `context` (REQ-30.15 de `project-technical-context`), `adopt` (REQ-30.9 y REQ-30.11 de `openspec-adoption`) e `integrate` (REQ-30.2 de `odd-living-spec-integration`), cuya ayuda se fija en esas capacidades.

## MODIFIED Requirements

### Requirement: Escaneo y Extracción de Especificaciones Vivas (REQ-1.1)

El indexador DEBE recorrer recursivamente el directorio `<planning home>/specs/` (`openspec/specs/`; el planning home lo resuelve el primitivo de la capacidad `planning-home-resolution`, REQ-24.1 y REQ-24.10), localizando todos los archivos `spec.md`. Para cada archivo DEBE extraer:
- Identificador de dominio (nombre de la subcarpeta).
- Título principal de la especificación (encabezado `# `).
- Propósito o descripción introductoria.
- Lista completa de requerimientos canónicos (`### Requirement: ...`).
- Lista de escenarios BDD vinculados (`#### Scenario: ...`).

Cuando en un proyecto `monorepo-embedded` con `specs_repository` distinto de `""`, `.` y `openspec` existe `<raíz>/<specs_repository>/specs` y difiere de `<planning home>/specs`, el indexador DEBE leer solo `<planning home>/specs` y emitir un aviso que nombre ambas rutas.
(Previously: además, con una baseline del proyecto declarada, el indexador no indexaba sus documentos e informaba de que era externa; la baseline ajena se retira)

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

---

### Requirement: Generación Determinista de `openspec/INDEX.md` (REQ-1.2)

El motor DEBE ser capaz de compilar y escribir un documento canónico `INDEX.md` en el planning home (`<planning home>/INDEX.md`, es decir, `openspec/INDEX.md`) que resuma el estado consolidado de la arquitectura y contratos vivos del sistema. El índice DEBE incluir:
1. Encabezado maestro y fecha/hora de la última sincronización.
2. Tabla resumen de especificaciones vivas con: Dominio, Título, Requerimientos, Escenarios y Enlace relativo al archivo `spec.md`.
3. Desglose por dominios con el listado de requerimientos canónicos y sus escenarios.

(Previously: con una baseline del proyecto declarada, el motor no creaba ni actualizaba `INDEX.md`; la baseline ajena se retira)

#### Scenario: Generación y persistencia de `openspec/INDEX.md`
- **DADO** un catálogo de especificaciones vivas indexado
- **CUANDO** el servicio ejecuta el método de sincronización de índice
- **ENTONCES** escribe el archivo `INDEX.md` en el planning home (`openspec/INDEX.md`) en el sistema de archivos
- **Y** todos los enlaces a las especificaciones son relativos y navegables

---

### Requirement: Síntesis de Especificación Viva a partir de un Cambio Archivado (REQ-2.1)

Cuando un incremento concluye en un proyecto donde el dominio correspondiente aún no cuenta con especificación viva en `<planning home>/specs/<dominio>/` (`openspec/specs/<dominio>/`), el sintetizador DEBE tomar los artefactos del incremento (`spec.md`, `proposal.md`, `verify-report.md`) y generar de forma determinista la especificación viva inicial en `<planning home>/specs/<dominio>/spec.md`.
(Previously: con una baseline del proyecto declarada, el sintetizador rechazaba la operación; la baseline ajena se retira)

#### Scenario: Adopción orgánica de un nuevo dominio
- **DADO** un cambio archivado o en fase de cierre para el dominio `billing-service`
- **Y** que no existe previamente `openspec/specs/billing-service/spec.md`
- **CUANDO** se invoca la operación de *Cold Start*
- **ENTONCES** se crea la carpeta `openspec/specs/billing-service/`
- **Y** se genera el archivo `spec.md` inicial con los requerimientos y contratos validados en el cambio

---

### Requirement: Comando `axiom archive sync` (REQ-3.1)

El comando `axiom archive sync [--path <dir>]` (donde `--path` es el directorio de partida de la resolución del planning home) DEBE:
1. Re-escanear todas las especificaciones vivas en `<planning home>/specs/` (`openspec/specs/`).
2. Generar y sobrescribir de forma determinista el archivo `<planning home>/INDEX.md` (`openspec/INDEX.md`).
3. Imprimir por consola el resumen con el total de especificaciones, requerimientos y escenarios sincronizados.

(Previously: con una baseline del proyecto declarada, el comando rechazaba la operación; la baseline ajena se retira)

#### Scenario: Sincronización exitosa desde CLI
- **DADO** un workspace de Axiom con especificaciones vivas
- **CUANDO** el usuario ejecuta `axiom archive sync`
- **ENTONCES** se actualiza `openspec/INDEX.md`
- **Y** la salida por consola muestra el informe de sincronización con código de salida 0

#### Scenario: Sincronización desde el workspace canónico
- **DADO** el workspace canónico con especificaciones vivas en `<maestra>/repo-specs/openspec/specs/`
- **CUANDO** el usuario ejecuta `axiom archive sync --path <maestra>`
- **ENTONCES** se actualiza `<maestra>/repo-specs/openspec/INDEX.md`

---

### Requirement: Comando `axiom archive coldstart` (REQ-3.4)

El comando `axiom archive coldstart --change <nombre> [--domain <dominio>] [--path <dir>]` DEBE promover un cambio verificado a especificación viva cuando no existía previamente, en el planning home resuelto.
(Previously: con una baseline del proyecto declarada, el comando rechazaba la operación; la baseline ajena se retira)

#### Scenario: Ejecución de coldstart para un cambio nuevo
- **DADO** un cambio completado con sus artefactos `spec.md` y `verify-report.md`
- **CUANDO** el usuario ejecuta `axiom archive coldstart --change 2026-09-14-inc-01-axiom-identity-workspace-topology`
- **ENTONCES** se consolida la especificación viva en `openspec/specs/`
- **Y** se ejecuta automáticamente la sincronización de `openspec/INDEX.md`

---

### Requirement: Endpoints REST para Especificaciones Vivas (REQ-4.1)

El servidor web DEBE exponer las siguientes rutas REST:
- `GET /api/archive/specs`: Retorna el catálogo completo `LivingCatalog` serializado en JSON.
- `GET /api/archive/specs/{domain}`: Retorna el contenido estructurado y texto Markdown de la spec viva del dominio solicitado.
- `POST /api/archive/sync`: Ejecuta la re-indexación y sincronización de `openspec/INDEX.md` desde la Web UI.

(Previously: con una baseline del proyecto declarada, `GET /api/archive/specs` respondía vacío con un indicador de baseline externa, `GET /api/archive/specs/{domain}` de un dominio declarado respondía 4xx y `POST /api/archive/sync` respondía 4xx; la baseline ajena se retira)

#### Scenario: Petición HTTP a `/api/archive/specs`
- **DADO** el servidor del Dashboard Web en ejecución
- **CUANDO** se efectúa una petición `GET` a `/api/archive/specs`
- **ENTONCES** responde con código HTTP 200 y cabecera `Content-Type: application/json`
- **Y** el cuerpo contiene la lista de especificaciones vivas con sus requerimientos

---

### Requirement: Interfaz Web SPA con Explorador de Documentación Viva (REQ-4.2)

La interfaz SPA embebida (`assets/index.html`, `app.js`, `style.css`) DEBE incluir una nueva pestaña de navegación **"Especificaciones Vivas"** que permita:
1. Visualizar tarjetas de resumen con el total de especificaciones vivas, requerimientos y escenarios BDD consolidados en el sistema.
2. Disponer de un botón de acción rápida para re-sincronizar el catálogo (`↻ Sincronizar Catálogo`).
3. Navegar interactivamente por los dominios de especificación y leer su contenido formateado con visor de requerimientos y escenarios.

(Previously: con una baseline del proyecto declarada, la pestaña mostraba un aviso de baseline externa y el botón de sincronización mostraba el rechazo; la baseline ajena se retira)

#### Scenario: Exploración de especificaciones vivas en el navegador
- **DADO** el usuario con el Dashboard abierto en la pestaña "Especificaciones Vivas"
- **CUANDO** selecciona una especificación de la lista
- **ENTONCES** se carga el detalle del dominio con su propósito, requerimientos y escenarios BDD
- **Y** se provee un botón para sincronizar el índice en un solo clic
