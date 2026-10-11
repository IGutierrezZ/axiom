# Especificación de Requerimientos: Resolución del Planning Home desde Cualquier Directorio (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Rebanada:** A (resolver de raíz) · **Tipo:** capacidad nueva
> **Idioma:** Español (castellano peninsular)

## Propósito

Definir cómo localiza Axiom el *planning home* (el directorio `openspec` que contiene los cambios SDD) desde cualquier directorio de partida de un workspace: la carpeta maestra, el repositorio de especificaciones o un repositorio de código. Un único primitivo de resolución sustituye a las tres convenciones incompatibles actuales: el motor SDD asume `<cwd>/openspec`; livingdoc, dashboard y skillregistry derivan `<specs_repository>/...`; y el perfil knowledge usa `specs_repository` relativo a su propio `axiom.yaml`.

**Resultado esperado:** desde cualquier directorio del workspace canónico, todos los verbos SDD operan sobre el mismo planning home; en `monorepo-embedded` el resultado es idéntico al actual.

## Términos y fixture de referencia

| Término | Definición |
|---|---|
| Directorio de partida | Directorio desde el que se inicia la resolución: `--cwd`, `--path` (verbos `handoff` y `role`) o el directorio de trabajo. |
| Carpeta maestra | Directorio que contiene `.axiom-workspace` o, sin puntero, el `axiom.yaml` del proyecto. Las rutas de `axiom.yaml` (`specs_repository`, repositorios de roles) son relativas a ella. |
| Repositorio de especificaciones | `<maestra>/<specs_repository>` en `monorepo-decoupled` y `multirepo`; en `monorepo-embedded`, la propia carpeta maestra. Su raíz, que es `workspaceRoot` y la que debe existir (REQ-24.7), es ese directorio o, si el último segmento de `specs_repository` es `openspec`, su padre. |
| Planning home | Directorio `openspec` que contiene `changes/` (y `specs/` en la baseline por defecto). |
| `workspaceRoot` | Directorio padre del planning home. Es el valor de `actionContext.workspaceRoot` en el status y la base contra la que se resuelven las rutas `openspec/...`. |
| Origen | Cómo se halló el workspace: `puntero`, `configuración`, `openspec` o `directorio de partida`. |

**Workspace canónico** (fixture que usan los escenarios): `<maestra>` (carpeta no versionada) con `.axiom-workspace` (`config: repo-specs/axiom.yaml`); `<maestra>/repo-specs` (repositorio git) con `axiom.yaml` (`topology: multirepo`, `specs_repository: repo-specs`) y `openspec/changes/`; y los repositorios de código hermanos `<maestra>/repo-backend` y `<maestra>/repo-frontend`.

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| REQ-24.1 | Primitivo único y de solo lectura | Una sola resolución, sin efectos en disco y con rechazos accionables |
| REQ-24.2 | Descubrimiento ascendente por proximidad | Gana el marcador más cercano al directorio de partida |
| REQ-24.3 | Enraizado por topología | `monorepo-decoupled` y `multirepo` re-enraízan en el repositorio de especificaciones |
| REQ-24.4 | Invariancia de `monorepo-embedded` | Resultado idéntico al actual: `<raíz>/openspec` |
| REQ-24.5 | Deducción de la carpeta maestra | Desde el repositorio de especificaciones se deduce la maestra; en `monorepo-embedded` es siempre el directorio del `axiom.yaml` |
| REQ-24.6 | Rechazo de la ambigüedad maestra con `openspec/` y puntero | Rechazo con continuación y sin escribir; `axiom init` no la produce |
| REQ-24.7 | Configuración no resoluble o no cargable | Con puntero, un destino roto o ilegible rechaza (planning home desconocido); sin puntero se conserva lo embebido con aviso |
| REQ-24.8 | Consumo idéntico por los verbos SDD | Mismo planning home desde cualquier punto de partida |
| REQ-24.9 | Semántica en el status y autoridad de edición | `workspaceRoot` y `planningHome` coherentes; repos de código por autorización |
| REQ-24.10 | Consumidores de catálogo | Una sola convención para livingdoc, knowledge, dashboard y TUI |
| REQ-24.11 | Regla de ejecutores | Las rutas `openspec/...` se resuelven contra `workspaceRoot` |
| REQ-24.12 | Documentación y evidencia `pr_merged` | Receta canónica documentada y `pr_merged` contra el repo de specs |

## No cubre esta capacidad

- Generar la topología canónica (`axiom adopt`): el puntero y el repositorio de especificaciones se montan a mano y se documentan (REQ-24.12).
- Dar semántica a `workspace.root`: el primitivo no lo lee (REQ-24.5).
- Un flag `--repo` para la evidencia `pr_merged`: queda para un cambio posterior.
- El descubrimiento de skills en `<specs_repository>/skills` y en los repositorios de roles.

---

## 1. Capacidad: `planning-home-resolution`

El primitivo de resolución determina carpeta maestra, `workspaceRoot`, planning home y origen a partir de un directorio de partida. Todos los verbos del flujo SDD y todos los consumidores de catálogo lo usan, de modo que un mismo workspace se resuelve igual desde cualquier punto.

### Requirement: Primitivo Único y de Solo Lectura de Resolución del Planning Home (REQ-24.1)

El sistema DEBE ofrecer un único primitivo de resolución que, dado un directorio de partida, devuelva la carpeta maestra, `workspaceRoot`, el planning home y el origen de la resolución. El primitivo DEBE ser determinista y de solo lectura: NO DEBE crear ni modificar ficheros ni directorios (tampoco `openspec/`). Toda resolución que no pueda completarse DEBE rechazarse nombrando la causa y una continuación accionable, sin escribir nada. Todos los consumidores de esta capacidad DEBEN obtener esos valores del primitivo y NO DEBEN mantener una convención propia.

#### Scenario: Resolución completa desde la carpeta maestra del workspace canónico
- **DADO** el workspace canónico
- **CUANDO** se resuelve desde `<maestra>`
- **ENTONCES** la carpeta maestra es `<maestra>`, `workspaceRoot` es `<maestra>/repo-specs`, el planning home es `<maestra>/repo-specs/openspec` y el origen es `puntero`

#### Scenario: La resolución no escribe en disco
- **DADO** el workspace canónico sin directorio `openspec` dentro de `repo-specs`
- **CUANDO** se resuelve desde cualquier directorio de partida del workspace
- **ENTONCES** el árbol de ficheros es idéntico antes y después de la resolución
- **Y** no se ha creado `openspec/` ni ningún fichero temporal

#### Scenario: Una resolución fallida no deja efectos y nombra la continuación
- **DADO** un `.axiom-workspace` cuyo `config` apunta a un fichero inexistente
- **CUANDO** se intenta resolver desde esa carpeta
- **ENTONCES** la resolución se rechaza con un mensaje que nombra el puntero, el destino inexistente y la acción para corregirlo
- **Y** el árbol de ficheros no cambia

---

### Requirement: Descubrimiento Ascendente por Proximidad (REQ-24.2)

Partiendo del directorio de partida, el primitivo DEBE evaluar primero ese directorio y después sus ancestros, y detenerse en el primero que contenga `.axiom-workspace`, un fichero `axiom.yaml` o un directorio `openspec`: gana el más cercano. Si un mismo directorio contiene varios marcadores, el origen se asigna por precedencia: `.axiom-workspace` (`puntero`), `axiom.yaml` (`configuración`) y `openspec` (`openspec`). El descubrimiento automático de `axiom.yaml` en los subdirectorios `repo-specs`, `specs`, `openspec`, `.openspec` y `especificacion` NO DEBE actuar como marcador ni mover el planning home. Si no existe ningún marcador hasta la raíz del sistema de ficheros, `workspaceRoot` DEBE ser el directorio de partida y el origen `directorio de partida`. Un directorio de partida inexistente, que no sea un directorio o que sea la raíz del sistema de ficheros DEBE rechazarse indicando que el directorio de partida (`--cwd` o `--path`) es incorrecto.

#### Scenario: Gana el directorio más cercano frente a un proyecto anidado
- **DADO** el workspace canónico donde `repo-backend` contiene además un directorio `openspec` propio
- **CUANDO** se resuelve desde `<maestra>/repo-backend/src`
- **ENTONCES** `workspaceRoot` es `<maestra>/repo-backend`, el planning home es `<maestra>/repo-backend/openspec` y el origen es `openspec`
- **Y** el puntero de `<maestra>` no interviene

#### Scenario: Un subdirectorio de un proyecto embebido resuelve la raíz del proyecto
- **DADO** un proyecto `P` con `axiom.yaml` (`monorepo-embedded`) y `P/openspec/`
- **CUANDO** se resuelve con `--cwd P/internal/pkg`
- **ENTONCES** `workspaceRoot` es `P` y el planning home es `P/openspec`
- **Y** el planning home no es `P/internal/pkg/openspec`

#### Scenario: El directorio de partida se evalúa antes que sus ancestros
- **DADO** `P` con `axiom.yaml` y un subdirectorio `P/sub` con su propio directorio `openspec`
- **CUANDO** se resuelve con `--cwd P/sub`
- **ENTONCES** `workspaceRoot` es `P/sub` y el origen es `openspec`

#### Scenario: Precedencia entre marcadores de un mismo directorio
- **DADO** un directorio que contiene `.axiom-workspace` y `axiom.yaml`
- **CUANDO** se resuelve desde él
- **ENTONCES** el origen es `puntero`
- **Y** la configuración efectiva es la que el puntero apunta

#### Scenario: Sin marcadores la raíz es el directorio de partida
- **DADO** un directorio sin marcadores en él ni en sus ancestros
- **CUANDO** se resuelve desde él
- **ENTONCES** `workspaceRoot` es ese directorio, el origen es `directorio de partida` y el planning home es `<directorio>/openspec`
- **Y** ese planning home no se crea

#### Scenario: El descubrimiento de subdirectorios no actúa como marcador
- **DADO** una carpeta `M` sin marcadores propios que contiene `repo-specs/axiom.yaml`
- **CUANDO** se resuelve desde `M`
- **ENTONCES** `workspaceRoot` es `M` y el origen es `directorio de partida`

#### Scenario: Directorio de partida inválido
- **DADO** un `--cwd` (o un `--path`) que no existe, que es un fichero o que es la raíz del sistema de ficheros
- **CUANDO** se resuelve
- **ENTONCES** se rechaza indicando que el directorio de partida es incorrecto
- **Y** el mensaje incluye el comando que lo corrige con el directorio del proyecto

---

### Requirement: Enraizado por Topología en Monorepo Desacoplado y Multirepo (REQ-24.3)

Una vez hallada la carpeta maestra (REQ-24.5) y cargado `axiom.yaml`, `workspaceRoot` y el planning home DEBEN derivarse según la topología:

- `monorepo-embedded`: `workspaceRoot` es la carpeta maestra y el planning home es `<workspaceRoot>/openspec` (REQ-24.4).
- `monorepo-decoupled` y `multirepo`: el repositorio de especificaciones es `<maestra>/<specs_repository>`. Si el último segmento de `specs_repository` es `openspec`, el planning home es ese directorio y `workspaceRoot` su padre. En otro caso, el planning home es `<repositorio de especificaciones>/openspec` y `workspaceRoot` es el repositorio de especificaciones.

En todos los casos `openspec/` DEBE permanecer literal bajo `workspaceRoot`, de modo que las rutas `openspec/changes/...` siguen siendo válidas respecto a `workspaceRoot`.

#### Scenario: Multirepo canónico
- **DADO** el workspace canónico
- **CUANDO** se resuelve desde `<maestra>`, desde `<maestra>/repo-specs` o desde `<maestra>/repo-backend/src`
- **ENTONCES** el planning home es `<maestra>/repo-specs/openspec` y `workspaceRoot` es `<maestra>/repo-specs` en los tres casos

#### Scenario: Monorepo desacoplado con `axiom.yaml` en la carpeta maestra
- **DADO** una carpeta maestra con `axiom.yaml` (`monorepo-decoupled`, `specs_repository: especificacion`) y los directorios `app-monorepo/` y `especificacion/`
- **CUANDO** se resuelve desde la carpeta maestra
- **ENTONCES** `workspaceRoot` es `<maestra>/especificacion` y el planning home es `<maestra>/especificacion/openspec`

#### Scenario: Perfil knowledge con `specs_repository` terminado en `openspec`
- **DADO** un directorio `X` con `axiom.yaml` (`multirepo`, `specs_repository: openspec`) y `X/openspec/`
- **CUANDO** se resuelve desde `X`
- **ENTONCES** el planning home es `X/openspec` y `workspaceRoot` es `X`

#### Scenario: `specs_repository` de varios segmentos terminado en `openspec`
- **DADO** una carpeta maestra con `axiom.yaml` (`multirepo`, `specs_repository: docs/openspec`)
- **CUANDO** se resuelve desde la carpeta maestra
- **ENTONCES** el planning home es `<maestra>/docs/openspec` y `workspaceRoot` es `<maestra>/docs`

---

### Requirement: Invariancia del Layout Monorepo Embebido (REQ-24.4)

En `monorepo-embedded`, y en un proyecto descubierto por `openspec` o por `directorio de partida` (sin `axiom.yaml`), el planning home DEBE ser exactamente `<raíz>/openspec`, con independencia del valor de `specs_repository` (`""`, `.`, `openspec`, `specs` u otro). Cuando el directorio de partida es la raíz del proyecto, `workspaceRoot` DEBE ser ese mismo directorio. La introducción del primitivo NO DEBE exigir modificar ninguna aserción de ruta de las pruebas existentes de `sddstatus`, `cli`, `kickoff` y `livingdoc`.

#### Scenario: Equivalencia con distintos valores de `specs_repository`
- **DADO** un proyecto `monorepo-embedded` y, para cada valor de `specs_repository` en {`""`, `.`, `openspec`, `specs`}, su `axiom.yaml` y un directorio `openspec/`
- **CUANDO** se ejecuta `axiom sdd status --json` desde la raíz del proyecto
- **ENTONCES** `actionContext.workspaceRoot` es la raíz del proyecto, `planningHome.path` es `<raíz>/openspec` y `planningHome.mode` es `repo-local`, idénticos para los cuatro valores

#### Scenario: Este repositorio no cambia
- **DADO** el repositorio de Axiom (`monorepo-embedded`, `specs_repository: "openspec"`, sin puntero)
- **CUANDO** se ejecuta `axiom sdd status --json` desde su raíz
- **ENTONCES** `actionContext.workspaceRoot` es la raíz del repositorio y `planningHome.path` es `<repo>/openspec`

#### Scenario: Proyecto con solo `openspec/` y sin configuración
- **DADO** un directorio con `openspec/` y sin `axiom.yaml` ni `.axiom-workspace`
- **CUANDO** se resuelve desde él
- **ENTONCES** `workspaceRoot` es ese directorio, el planning home es `<directorio>/openspec` y el origen es `openspec`

#### Scenario: Puntero que apunta al propio `axiom.yaml` embebido
- **DADO** un proyecto con `.axiom-workspace` (`config: axiom.yaml`), `axiom.yaml` con `monorepo-embedded` y `openspec/`
- **CUANDO** se resuelve desde la raíz
- **ENTONCES** el planning home es `<raíz>/openspec` y el origen es `puntero`

---

### Requirement: Deducción de la Carpeta Maestra desde el Repositorio de Especificaciones (REQ-24.5)

Cuando el marcador hallado es un `axiom.yaml` sin puntero en el directorio `D`, la carpeta maestra DEBE determinarse según la topología. En `monorepo-embedded` es siempre `D`: la deducción por sufijo no se aplica, porque en un proyecto embebido cuyo directorio se llame igual que `specs_repository` (por ejemplo `W/specs` con `specs_repository: specs`) movería `workspaceRoot` al directorio padre y rompería la invariancia exacta de REQ-24.4. En `monorepo-decoupled` y `multirepo` se determina por este orden: (1) si `specs_repository` es vacío o `.`, `D`; (2) si `D` contiene un directorio con la ruta `specs_repository`, `D`; (3) si la ruta de `D` termina con los segmentos de `specs_repository`, `D` sin ese sufijo; (4) en otro caso, `D`. La comparación de segmentos DEBE tratar `/` y `\` como equivalentes. Cuando el marcador es un puntero, la carpeta maestra es el directorio que lo contiene. El campo `workspace.root` NO DEBE intervenir en la resolución.

#### Scenario: Desde el repositorio de especificaciones se deduce la carpeta maestra
- **DADO** el workspace canónico
- **CUANDO** se resuelve desde `<maestra>/repo-specs`
- **ENTONCES** la carpeta maestra es `<maestra>`
- **Y** el planning home y `workspaceRoot` coinciden con los obtenidos desde `<maestra>`

#### Scenario: `specs_repository` de varios segmentos
- **DADO** un `axiom.yaml` `multirepo` en `<maestra>/repos/specs` con `specs_repository: repos/specs`
- **CUANDO** se resuelve desde `<maestra>/repos/specs`
- **ENTONCES** la carpeta maestra es `<maestra>`

#### Scenario: En `monorepo-embedded` no se aplica la deducción por sufijo
- **DADO** un proyecto `monorepo-embedded` cuyo directorio es `W/specs`, con `axiom.yaml` (`specs_repository: specs`) y `W/specs/openspec/`
- **CUANDO** se resuelve desde `W/specs`
- **ENTONCES** la carpeta maestra y `workspaceRoot` son `W/specs` y el planning home es `W/specs/openspec`
- **Y** la carpeta maestra no es `W`

#### Scenario: El directorio que contiene `specs_repository` es la carpeta maestra
- **DADO** un directorio `W/openspec` con `axiom.yaml` (`multirepo`, `specs_repository: openspec`) que contiene a su vez `W/openspec/openspec/`
- **CUANDO** se resuelve desde `W/openspec`
- **ENTONCES** la carpeta maestra es `W/openspec`
- **Y** el planning home es `W/openspec/openspec`

#### Scenario: Sin coincidencia de sufijo la carpeta maestra es el directorio del `axiom.yaml`
- **DADO** un `axiom.yaml` en `D` con `specs_repository: especificacion` y un directorio `D/especificacion`
- **CUANDO** se resuelve desde `D`
- **ENTONCES** la carpeta maestra es `D`

#### Scenario: `workspace.root` no altera la resolución
- **DADO** un proyecto `monorepo-embedded` cuyo `axiom.yaml` declara `workspace.root: ".."`
- **CUANDO** se resuelve desde su raíz
- **ENTONCES** `workspaceRoot` y el planning home son los mismos que sin ese campo

---

### Requirement: Rechazo de la Ambigüedad de Carpeta Maestra con `openspec/` Propio y Puntero (REQ-24.6)

Cuando la carpeta maestra contiene un puntero `.axiom-workspace` y además un directorio `openspec` propio (aunque esté vacío), y el planning home que resulta de la configuración apuntada es distinto de `<maestra>/openspec`, el primitivo DEBE rechazar la resolución. El rechazo DEBE nombrar ambos directorios y la continuación (consolidar los cambios de `<maestra>/openspec` en el planning home resuelto y retirar el directorio sobrante, o retirar el puntero), NO DEBE escribir ni mover nada y DEBE aplicarse a todos los verbos que consumen el primitivo, también cuando la resolución parte de un subdirectorio. Ningún creador DEBE producir esta ambigüedad: `axiom init` se rechaza en una carpeta maestra con puntero hacia otro planning home, sin escribir nada (REQ-2.2 de la capacidad `multi-project-hub`).

#### Scenario: Ambigüedad rechazada con continuación
- **DADO** el workspace canónico más un directorio `<maestra>/openspec/changes/cambio-antiguo`
- **CUANDO** se ejecuta `axiom sdd status --cwd <maestra>`
- **ENTONCES** el comando se rechaza con código de salida distinto de cero
- **Y** el mensaje nombra `<maestra>/openspec`, `<maestra>/repo-specs/openspec` y la acción para consolidarlos o retirar el puntero
- **Y** no se escribe ni se mueve ningún fichero

#### Scenario: La ambigüedad se detecta también desde un repositorio de código
- **DADO** el mismo workspace con el directorio `<maestra>/openspec`
- **CUANDO** se resuelve desde `<maestra>/repo-backend/src`
- **ENTONCES** se rechaza con el mismo mensaje

#### Scenario: Mismo planning home no es ambiguo
- **DADO** una carpeta maestra `M` con `.axiom-workspace` (`config: axiom.yaml`), `axiom.yaml` (`multirepo`, `specs_repository: openspec`) y `M/openspec/`
- **CUANDO** se resuelve desde `M`
- **ENTONCES** se resuelve sin rechazo con planning home `M/openspec`

#### Scenario: Sin puntero no hay ambigüedad
- **DADO** un directorio con `axiom.yaml` (sin puntero) y `openspec/`
- **CUANDO** se resuelve desde él
- **ENTONCES** se resuelve sin rechazo

---

### Requirement: Configuración No Resoluble o No Cargable (REQ-24.7)

El primitivo DEBE distinguir cuatro casos:

1. Un puntero cuyo destino no existe, o que no define `config` ni `specs`, DEBE rechazarse nombrando el puntero, el destino y la acción para corregirlo o retirarlo.
2. Un puntero cuyo destino existe pero cuyo `axiom.yaml` no puede leerse como YAML o no supera la validación de la configuración base (nombre, topología y roles) DEBE rechazarse nombrando la causa y `axiom workspace validate` como continuación: la configuración base determina el repositorio de especificaciones, de modo que el planning home es desconocido y no hay resolución ni status que devolver. Un destino legible cuya configuración base es válida y cuyo único defecto es un bloque `baseline:` o `technical_context:` inválido NO DEBE rechazarse: el planning home es determinable y se aplica REQ-24.18.
3. Un `axiom.yaml` directo (sin puntero) que no puede cargarse NO DEBE rechazarse: el primitivo DEBE conservar la resolución embebida (`workspaceRoot` es el directorio que lo contiene y el planning home es `<raíz>/openspec`) y comunicar un aviso no bloqueante que nombre la causa y `axiom workspace validate`. Que esa configuración impida declarar una baseline (fichero ilegible o bloque inválido) deja el status operativo y bloquea solo el archivado (REQ-24.18).
4. En `monorepo-decoupled` y `multirepo`, una raíz del repositorio de especificaciones (`workspaceRoot`) inexistente DEBE rechazarse nombrando la ruta esperada y la acción para crearla o corregir `specs_repository`. Esa raíz es `<maestra>/<specs_repository>` o, si el último segmento de `specs_repository` es `openspec`, su directorio padre. El planning home (`openspec/`) NO tiene que existir para resolver: el primitivo no lo exige ni lo crea (REQ-24.1).

#### Scenario: Puntero sin `config` ni `specs`
- **DADO** un `.axiom-workspace` vacío
- **CUANDO** se resuelve desde su carpeta
- **ENTONCES** se rechaza indicando que el puntero debe definir `config` o `specs`

#### Scenario: Destino del puntero con configuración inválida
- **DADO** un `.axiom-workspace` cuyo destino es un `axiom.yaml` con `workspace.topology: "desconocida"`
- **CUANDO** se resuelve desde la carpeta del puntero
- **ENTONCES** se rechaza nombrando la topología inválida y `axiom workspace validate`

#### Scenario: Puntero hacia un `axiom.yaml` que no es YAML válido
- **DADO** un `.axiom-workspace` cuyo destino es un `axiom.yaml` con error de sintaxis YAML
- **CUANDO** se ejecuta `axiom sdd status --json` desde la carpeta del puntero
- **ENTONCES** se rechaza con código de salida distinto de cero nombrando la causa y `axiom workspace validate`, porque el planning home es desconocido
- **Y** no se devuelve ningún status y no se escribe nada

#### Scenario: Puntero hacia una configuración legible con un bloque `baseline:` inválido
- **DADO** un `.axiom-workspace` cuyo destino es un `axiom.yaml` con configuración base válida y con `baseline.documents[0].requirement.identity: nombre`
- **CUANDO** se ejecuta `axiom sdd status --json` desde la carpeta del puntero
- **ENTONCES** la resolución se completa sin rechazo y el status devuelve el planning home que determina el puntero
- **Y** el status señala la causa y marca el archivado como bloqueado (REQ-24.18)

#### Scenario: `axiom.yaml` directo no cargable conserva la resolución embebida
- **DADO** un proyecto con `openspec/` y un `axiom.yaml` sin la sección `roles`
- **CUANDO** se ejecuta `axiom sdd status --json` desde su raíz
- **ENTONCES** `workspaceRoot` es la raíz del proyecto y `planningHome.path` es `<raíz>/openspec`
- **Y** el status incluye un aviso no bloqueante que nombra la causa y `axiom workspace validate`

#### Scenario: Repositorio de especificaciones declarado e inexistente
- **DADO** una carpeta maestra con `axiom.yaml` (`multirepo`, `specs_repository: repo-specs`) y sin el directorio `repo-specs`
- **CUANDO** se resuelve desde la carpeta maestra
- **ENTONCES** se rechaza nombrando `<maestra>/repo-specs` y la acción para crearlo o corregir `specs_repository`

#### Scenario: `specs_repository` terminado en `openspec` con el planning home aún sin crear
- **DADO** una carpeta maestra `M` con `axiom.yaml` (`multirepo`, `specs_repository: openspec`) y sin el directorio `M/openspec`
- **CUANDO** se resuelve desde `M`
- **ENTONCES** no se rechaza: `workspaceRoot` es `M` y el planning home es `M/openspec`
- **Y** `M/openspec` no se crea

---

### Requirement: Consumo Idéntico por los Verbos del Flujo SDD (REQ-24.8)

Los siguientes verbos DEBEN obtener carpeta maestra, `workspaceRoot`, planning home y configuración del primitivo de REQ-24.1: `axiom sdd status`, `axiom sdd continue`, `axiom sdd kickoff seal|show`, `axiom sdd gate record|show`, `axiom sdd-attempt grant`, `axiom sdd archive-compose` (cuando recibe `--cwd`), `axiom handoff show|create|validate` y `axiom role list|status|barrier`. Desde un mismo workspace DEBEN resolver el mismo planning home con independencia del directorio de partida. La configuración de `axiom.yaml` que estos verbos consultan (roles, topología) DEBE cargarse a través del resultado de la resolución y NO comprobando la existencia de `<raíz>/axiom.yaml`, de modo que un puntero sin `axiom.yaml` en la carpeta maestra se honre.

#### Scenario: Mismo planning home desde tres puntos de partida
- **DADO** el workspace canónico
- **CUANDO** se ejecuta `axiom sdd status --json` desde `<maestra>`, desde `<maestra>/repo-specs` y desde `<maestra>/repo-backend/src`
- **ENTONCES** los tres devuelven `planningHome.path` igual a `<maestra>/repo-specs/openspec` y `actionContext.workspaceRoot` igual a `<maestra>/repo-specs`

#### Scenario: Todos los verbos operan sobre el mismo cambio
- **DADO** el workspace canónico con el cambio `inc-99-ejemplo` en `<maestra>/repo-specs/openspec/changes/`
- **CUANDO** se ejecutan `kickoff seal|show`, `gate record|show`, `handoff validate` y `role barrier` para ese cambio desde `<maestra>` y desde `<maestra>/repo-backend/src`
- **ENTONCES** todos leen y escriben el cambio en `<maestra>/repo-specs/openspec/changes/inc-99-ejemplo`
- **Y** ninguno crea `openspec/` fuera del repositorio de especificaciones

#### Scenario: La configuración apuntada por el puntero se honra sin `axiom.yaml` en la carpeta maestra
- **DADO** el workspace canónico, sin `axiom.yaml` en `<maestra>`, con roles declarados en `repo-specs/axiom.yaml`
- **CUANDO** se ejecuta `axiom role list --change inc-99-ejemplo --path <maestra>`
- **ENTONCES** el roster se construye con los roles de `repo-specs/axiom.yaml`

#### Scenario: Directorio de partida de los verbos con `--path`
- **DADO** el workspace canónico
- **CUANDO** se ejecuta `axiom handoff show --change inc-99-ejemplo --path <maestra>/repo-backend`
- **ENTONCES** el handoff se busca en el planning home del workspace canónico

---

### Requirement: Semántica de `workspaceRoot`, `planningHome` y Autoridad de Edición en el Status (REQ-24.9)

El status v2 DEBE reportar `actionContext.workspaceRoot` igual al `workspaceRoot` resuelto, `planningHome.path` igual a la ruta absoluta del directorio `openspec` resuelto, `planningHome.mode` igual a `repo-local`, y `changeRoot` y `artifactPaths` bajo `planningHome.path`. `actionContext.allowedEditRoots` DEBE partir de `workspaceRoot`; los repositorios de código hermanos NO DEBEN autorizarse implícitamente y siguen el flujo de autorización por cambio existente (`axiom sdd-attempt grant`). Las autorizaciones por cambio, el registro de ejecución y el rechazo de cambios archivados DEBEN operar respecto a `workspaceRoot` y al repositorio git que lo contiene.

#### Scenario: Valores del status en el workspace canónico
- **DADO** el workspace canónico con el cambio `inc-99-ejemplo`
- **CUANDO** se ejecuta `axiom sdd status inc-99-ejemplo --json` desde `<maestra>`
- **ENTONCES** `planningHome.path` es `<maestra>/repo-specs/openspec`, `planningHome.mode` es `repo-local` y `changeRoot` es `<maestra>/repo-specs/openspec/changes/inc-99-ejemplo`
- **Y** `actionContext.allowedEditRoots` es `["<maestra>/repo-specs"]`

#### Scenario: Un repositorio de código hermano exige autorización por cambio
- **DADO** un `tasks.md` cuya unidad de trabajo nombra un fichero de `<maestra>/repo-backend`
- **CUANDO** se consulta el status del cambio
- **ENTONCES** `applyState` es `blocked` con un motivo `edit_authority_missing` que nombra `<maestra>/repo-backend`
- **Y** el repositorio no figura en `allowedEditRoots` hasta que se ejecuta `axiom sdd-attempt grant`

#### Scenario: Los cambios archivados siguen sellados
- **DADO** un cambio en `<maestra>/repo-specs/openspec/changes/archive/2026-10-04-inc-98-previo`
- **CUANDO** se ejecuta `axiom sdd gate record --cwd <maestra> --change inc-98-previo --gate spec --decision approved`
- **ENTONCES** la operación se rechaza por tratarse de un cambio archivado

---

### Requirement: Los Consumidores de Catálogo Comparten la Convención de Raíz (REQ-24.10)

El servicio de documentación viva (livingdoc, incluido el sintetizador de arranque en frío), knowledge (raíz de especificaciones por defecto), el dashboard web y la TUI DEBEN obtener la raíz de especificaciones y el planning home del primitivo de REQ-24.1 y NO DEBEN mantener una convención propia. Las especificaciones vivas, los cambios y el índice residen en `<planning home>/specs`, `<planning home>/changes` y `<planning home>/INDEX.md`. Cuando en un proyecto `monorepo-embedded` con `specs_repository` distinto de `""`, `.` y `openspec` existe `<raíz>/<specs_repository>/specs` y difiere de `<planning home>/specs`, los consumidores de catálogo DEBEN leer solo `<planning home>/specs` y emitir un aviso que nombre ambas rutas.

#### Scenario: Catálogo del workspace canónico
- **DADO** el workspace canónico con `<maestra>/repo-specs/openspec/specs/workspace-topology/spec.md`
- **CUANDO** se ejecuta `axiom archive list` desde `<maestra>`
- **ENTONCES** el listado incluye el dominio `workspace-topology`

#### Scenario: Raíz de especificaciones por defecto de knowledge
- **DADO** el workspace canónico
- **CUANDO** se ejecuta `axiom knowledge sweep` sin indicar raíz de especificaciones desde `<maestra>`
- **ENTONCES** la raíz de especificaciones usada es el planning home `<maestra>/repo-specs/openspec`

#### Scenario: Ubicación divergente en un proyecto embebido
- **DADO** un proyecto `monorepo-embedded` con `specs_repository: specs`, un `specs/specs/antiguo/spec.md` y un `openspec/specs/vigente/spec.md`
- **CUANDO** se ejecuta `axiom archive list`
- **ENTONCES** el listado incluye `vigente` y no incluye `antiguo`
- **Y** se emite un aviso que nombra `specs/specs` y `openspec/specs`

#### Scenario: Dashboard y TUI leen el mismo planning home
- **DADO** el dashboard web con el workspace canónico activo (ruta registrada `<maestra>`)
- **CUANDO** se consultan `GET /api/increments` y `GET /api/archive/specs`
- **ENTONCES** ambas respuestas proceden de `<maestra>/repo-specs/openspec`
- **Y** la pantalla de ciclo de vida de incrementos de la TUI lista los cambios de `<maestra>/repo-specs/openspec/changes`

---

### Requirement: Regla de Ejecutores para las Rutas `openspec/...` (REQ-24.11)

La convención compartida de las skills SDD (`_shared/sdd-phase-common.md`) DEBE indicar que las rutas `openspec/...` que citan las skills se resuelven contra `actionContext.workspaceRoot` del status y no contra el directorio de trabajo, y que los bloques de shell que operan sobre ellas DEBEN situarse primero en `workspaceRoot`. Los comandos de orquestador existentes que derivan `--cwd` del repositorio git contenedor DEBEN seguir resolviendo el mismo planning home sin cambios.

#### Scenario: Regla presente en la convención compartida
- **DADO** el texto de `_shared/sdd-phase-common.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** contiene la regla de resolver las rutas `openspec/...` contra `actionContext.workspaceRoot`

#### Scenario: Comando existente lanzado desde un repositorio de código
- **DADO** el workspace canónico y un comando de orquestador que pasa como `--cwd` la raíz git de `<maestra>/repo-backend`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el planning home es `<maestra>/repo-specs/openspec`
- **Y** el texto del comando no requiere modificación

---

### Requirement: Documentación de la Receta Canónica y de la Evidencia `pr_merged` (REQ-24.12)

`docs/manual-de-inicio.md` DEBE documentar la receta multirepo canónica: carpeta maestra no versionada con `.axiom-workspace` que apunta a `<specs_repository>/axiom.yaml`, repositorio de especificaciones y repositorios de código hermanos, indicando que se monta a mano. DEBE documentar además que `axiom sdd gate record --evidence-kind pr_merged` se verifica contra el grafo git del repositorio de especificaciones (`workspaceRoot`) y que la integración de los repositorios de código se acredita con las evidencias `deployment` o `attestation`.

#### Scenario: Receta canónica documentada
- **DADO** el texto de `docs/manual-de-inicio.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** describe `.axiom-workspace`, `<specs_repository>/axiom.yaml` y el montaje manual de la topología

#### Scenario: `pr_merged` se verifica contra el repositorio de especificaciones
- **DADO** el workspace canónico, un commit `S` fusionado en la rama principal de `repo-specs` y un commit `C` presente solo en `repo-backend`
- **CUANDO** se ejecuta `axiom sdd gate record --cwd <maestra> --change inc-99-ejemplo --gate integration --decision approved --evidence-kind pr_merged --commit S`
- **ENTONCES** la evidencia se acepta
- **Y** con `--commit C` la evidencia se rechaza por no estar en el grafo del repositorio de especificaciones

#### Scenario: Evidencia alternativa para repositorios de código
- **DADO** el texto de `docs/manual-de-inicio.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica que la integración de un repositorio de código se acredita con `deployment` o `attestation`
