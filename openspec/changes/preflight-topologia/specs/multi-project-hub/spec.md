# Delta para multi-project-hub

> Parte del estado vigente: el resto de requisitos NO cambia. En `monorepo-embedded` el alta solo escribe el rol `fullstack` (REQ-2.2 de `workspace-topology`); los roles detectados (`core`, `web`, `qa`) solo se aplican en `multirepo`.

## MODIFIED Requirements

### Requirement: Detección Heurística de Tecnologías (REQ-2.1)

El inicializador DEBE inspeccionar los archivos del repositorio para deducir la tecnología principal y configurar adecuadamente las herramientas y, solo en `multirepo`, los roles en `axiom.yaml` (en `monorepo-embedded` el único rol que se escribe es `fullstack`, REQ-2.2):
- Si detecta `go.mod` ➔ Tecnología: `go` / `go-test`.
- Si detecta `package.json` ➔ Tecnología: `nodejs` / `typescript` (o `react`/`vue`/`svelte` si están en dependencias).
- Si detecta `*.csproj` o `*.sln` ➔ Tecnología: `dotnet` / `csharp`.
- Si detecta `Cargo.toml` ➔ Tecnología: `rust`.
- Si detecta `pyproject.toml` o `requirements.txt` ➔ Tecnología: `python`.

#### Scenario: Detección automática en proyecto Go
- **DADO** un directorio con un archivo `go.mod`
- **CUANDO** se ejecuta el detector tecnológico
- **ENTONCES** el detector retorna `go` como tecnología base del rol `core` y `go-test` para el rol `qa`
- **Y** esos roles detectados solo se escriben en `axiom.yaml` si la topología es `multirepo`

#### Scenario: Detección automática en proyecto .NET / C#
- **DADO** un directorio con un archivo `Ludeka.sln` o `*.csproj`
- **CUANDO** se ejecuta el detector tecnológico
- **ENTONCES** retorna `csharp` y `dotnet` en la configuración inicial


### Requirement: Generación de Andamiaje `axiom.yaml` y Registro Automático (REQ-2.2)

Al ejecutar `axiom init` sobre un directorio:
1. Si NO existe `axiom.yaml`, genera un archivo canónico con el nombre del proyecto, topología seleccionada (`monorepo-embedded` por defecto), repositorio de especificaciones (`openspec`), roles y gobernanza en español. Los roles dependen de la topología: en `monorepo-embedded` el alta escribe SOLO el rol `fullstack`, aunque el detector haya deducido otros roles o el formulario los haya enviado; en `multirepo` se aplican los roles detectados (`core`, `web`, `qa`) o los del formulario. El perfil `knowledge` (o `spec-only`) fuerza `multirepo` con un único rol `knowledge`.
2. Crea las carpetas base necesarias: `.axiom/inbox/skills/`, `openspec/specs/` y `openspec/changes/`.
3. Registra automáticamente el proyecto en `~/.axiom/workspaces.json` y lo marca como proyecto activo.
4. Si YA existe `axiom.yaml`, no lo sobreescribe; valida su estructura (incluido el bloque opcional `technical_context`; un bloque `baseline:` residual se rechaza con el error tipado de REQ-30.21) y lo registra en el Hub global si no estaba registrado.

Antes de escribir nada, `axiom init` DEBE rechazarse cuando el propio directorio contiene un puntero `.axiom-workspace` cuyo planning home, resuelto con el primitivo de la capacidad `planning-home-resolution` (REQ-24.1), es distinto de `<directorio>/openspec`: crear ese directorio produciría un segundo planning home y la ambigüedad de REQ-24.6 en todos los verbos. La comparación es entre rutas y NO exige que `<directorio>/openspec` exista, porque `init` es quien lo crea; la existencia del repositorio de especificaciones que exige REQ-24.7 (caso 4) es una condición del primitivo y no de esta comparación. El rechazo DEBE aplicarse con cualquier perfil, finalizar con código de salida distinto de cero, no escribir ni registrar nada y nombrar el puntero, el planning home resuelto y la continuación: inicializar el `workspaceRoot` resuelto con `axiom init --path <workspaceRoot>` o montar la topología a mano según la receta documentada (REQ-24.12).

Cuando el puntero del propio directorio no puede resolverse (por ejemplo, un puntero roto o un `axiom.yaml` apuntado ilegible, REQ-24.7), el primitivo rechaza la resolución y `axiom init` DEBE rechazarse con ese mismo error, que nombra la causa y la continuación, con código de salida distinto de cero y sin escribir ni registrar nada.

`axiom init` NO DEBE rechazarse cuando el puntero resuelve a `<directorio>/openspec` (exista ya ese directorio o no), cuando el propio directorio no contiene puntero (aunque lo contenga un directorio ancestro: el marcador más cercano gana, REQ-24.2) ni cuando no hay ningún marcador: en esos casos se comporta como describen los puntos 1 a 4.

(Previously: el punto 2 omitía `openspec/specs/` e `openspec/INDEX.md` cuando `axiom.yaml` declaraba una baseline del proyecto y el punto 4 validaba también el bloque `baseline`; la baseline ajena se retira y `openspec/specs/` se crea siempre; el andamiaje escribía los roles detectados `core` y `qa` en cualquier topología, y ahora en `monorepo-embedded` escribe solo `fullstack`)

#### Scenario: Inicialización exitosa de proyecto virgen
- **DADO** un directorio limpio sin archivos de Axiom
- **CUANDO** se ejecuta `axiom init --name "MiProyecto"`
- **ENTONCES** se crea `axiom.yaml` con la estructura canónica
- **Y** se crean las carpetas `openspec/specs/`, `openspec/changes/` y `.axiom/inbox/skills/`
- **Y** el proyecto queda registrado en `~/.axiom/workspaces.json`

#### Scenario: Inicialización sobre un `axiom.yaml` con `baseline:` residual
- **DADO** un directorio con un `axiom.yaml` que declara `baseline:` con al menos un documento
- **CUANDO** se ejecuta `axiom init`
- **ENTONCES** el comando se rechaza con el error tipado de REQ-30.21 y la continuación `axiom archive adopt`
- **Y** el `axiom.yaml` existente no se sobreescribe ni se registra el proyecto

#### Scenario: Inicialización rechazada en una carpeta maestra con puntero hacia otro planning home
- **DADO** el workspace canónico de la capacidad `planning-home-resolution`, cuya carpeta maestra `<maestra>` contiene `.axiom-workspace` (`config: repo-specs/axiom.yaml`) y no contiene `openspec/`
- **CUANDO** se ejecuta `axiom init --path <maestra>`, con o sin `--profile=knowledge`
- **ENTONCES** el comando se rechaza con código de salida distinto de cero
- **Y** el mensaje nombra el puntero, el planning home resuelto `<maestra>/repo-specs/openspec` y la continuación `axiom init --path <maestra>/repo-specs` o la receta manual documentada
- **Y** no se crea `<maestra>/openspec/`, ni `axiom.yaml`, ni `.axiom/inbox/skills/`, y `~/.axiom/workspaces.json` no se modifica

#### Scenario: Un puntero propio que no puede resolverse rechaza la inicialización
- **DADO** un directorio con un `.axiom-workspace` cuyo `config` apunta a un fichero inexistente, o a un `axiom.yaml` que no es YAML válido
- **CUANDO** se ejecuta `axiom init --path <directorio>`
- **ENTONCES** el comando se rechaza con código de salida distinto de cero y con el error del primitivo de resolución, que nombra la causa y la continuación
- **Y** no se crea ningún fichero ni directorio y `~/.axiom/workspaces.json` no se modifica

#### Scenario: Un puntero que resuelve a `<directorio>/openspec` no se rechaza aunque ese directorio no exista todavía
- **DADO** una carpeta `M` con `.axiom-workspace` (`config: axiom.yaml`), un `axiom.yaml` `monorepo-embedded` y sin el directorio `M/openspec`
- **CUANDO** se ejecuta `axiom init --path M`
- **ENTONCES** no se rechaza, porque el planning home que resuelve el puntero es `M/openspec` y la comparación no exige que exista
- **Y** la inicialización se comporta como describen los puntos 1 a 4 y crea `M/openspec/`

#### Scenario: Un puntero en un directorio ancestro no impide inicializar
- **DADO** el mismo workspace canónico y el directorio `<maestra>/repo-backend`, que no contiene `.axiom-workspace` propio
- **CUANDO** se ejecuta `axiom init --path <maestra>/repo-backend`
- **ENTONCES** no se rechaza por el puntero de `<maestra>` y la inicialización se comporta como antes de este incremento


#### Scenario: El alta en monorepo escribe solo `fullstack` aunque se detecten otros roles
- **DADO** un directorio con `go.mod` y `package.json` cuyo detector deduce los roles `core`, `web` y `qa`
- **CUANDO** se ejecuta `axiom init` con la topología `monorepo-embedded` (por defecto)
- **ENTONCES** el `axiom.yaml` generado declara exactamente un rol, `fullstack`
- **Y** `axiom workspace validate` sobre ese `axiom.yaml` no reporta errores de roles

#### Scenario: El alta en monorepo ignora los roles enviados por el formulario
- **DADO** un alta con la topología `monorepo-embedded` y una lista de roles `core` y `qa` en los parámetros
- **CUANDO** se genera el `axiom.yaml`
- **ENTONCES** el `axiom.yaml` declara exactamente un rol, `fullstack`

#### Scenario: El alta en multirepo aplica los roles detectados o del formulario
- **DADO** un alta con la topología `multirepo` y los roles `core` y `qa` (detectados o enviados)
- **CUANDO** se genera el `axiom.yaml`
- **ENTONCES** el `axiom.yaml` declara los roles `core` y `qa`

#### Scenario: El perfil `knowledge` sigue forzando `multirepo` con el rol `knowledge`
- **DADO** un alta con el perfil `knowledge` (o `spec-only`) y cualquier topología solicitada
- **CUANDO** se genera el `axiom.yaml`
- **ENTONCES** la topología es `multirepo`
- **Y** el único rol declarado es `knowledge`

## ADDED Requirements

### Requirement: El Formulario de Alta del Dashboard Solo Ofrece Roles en `multirepo` (REQ-3.3)

El formulario de alta de proyecto del dashboard DEBE ofrecer el constructor de roles únicamente cuando la topología seleccionada es `multirepo`. Con `monorepo-embedded` NO DEBE mostrar el constructor de roles y el rol enviado DEBE ser siempre `fullstack`. Al cambiar la topología del formulario, el constructor DEBE mostrarse u ocultarse en consecuencia y la validación del formulario NO DEBE exigir roles especializados en monorepo.

#### Scenario: Con monorepo no hay constructor de roles y el rol es `fullstack`
- **DADO** el formulario de alta con la topología `monorepo-embedded`
- **CUANDO** el usuario lo envía
- **ENTONCES** el constructor de roles no se muestra
- **Y** los roles enviados son exactamente `fullstack`

#### Scenario: Con multirepo el constructor de roles está disponible
- **DADO** el formulario de alta con la topología `multirepo`
- **CUANDO** el usuario lo abre
- **ENTONCES** el constructor de roles se muestra y los roles que defina son los que se envían

#### Scenario: Cambiar de multirepo a monorepo descarta los roles especializados
- **DADO** el formulario con la topología `multirepo` y los roles `core` y `qa` definidos
- **CUANDO** el usuario cambia la topología a `monorepo-embedded` y lo envía
- **ENTONCES** los roles enviados son exactamente `fullstack`
