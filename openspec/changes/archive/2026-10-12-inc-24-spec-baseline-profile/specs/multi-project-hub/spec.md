# Delta para Hub Multi-Proyecto, Selector Dinámico en Dashboard Web y CLI axiom init (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Tipo:** delta sobre la capacidad viva `multi-project-hub`

Cambios de este delta:

- `axiom init` no crea `openspec/specs/` ni `openspec/INDEX.md` cuando `axiom.yaml` ya declara una baseline del proyecto (REQ-24.23), de modo que nunca produce una segunda baseline.
- `axiom init` se rechaza sin escribir nada cuando el propio directorio contiene un puntero `.axiom-workspace` hacia otro planning home, de modo que nunca produce un segundo planning home (REQ-24.6), o un puntero que no puede resolverse (REQ-24.7).

El resto del andamiaje y el registro en el Hub no cambian.

## MODIFIED Requirements

### Requirement: Generación de Andamiaje `axiom.yaml` y Registro Automático (REQ-2.2)

Al ejecutar `axiom init` sobre un directorio:
1. Si NO existe `axiom.yaml`, genera un archivo canónico con el nombre del proyecto, topología seleccionada (`monorepo-embedded`), repositorio de especificaciones (`openspec`), roles detectados (`core` y `qa`) y gobernanza en español.
2. Crea las carpetas base necesarias: `.axiom/inbox/skills/`, `openspec/specs/` y `openspec/changes/`, salvo que `axiom.yaml` ya declare una baseline del proyecto (`baseline:`): en ese caso NO crea `openspec/specs/` ni `openspec/INDEX.md` (tampoco con el perfil `knowledge`), informa de la omisión y continúa con el resto del andamiaje (REQ-24.23).
3. Registra automáticamente el proyecto en `~/.axiom/workspaces.json` y lo marca como proyecto activo.
4. Si YA existe `axiom.yaml`, no lo sobreescribe; valida su estructura (incluidos los bloques opcionales `baseline` y `technical_context`) y lo registra en el Hub global si no estaba registrado.

Antes de escribir nada, `axiom init` DEBE rechazarse cuando el propio directorio contiene un puntero `.axiom-workspace` cuyo planning home, resuelto con el primitivo de la capacidad `planning-home-resolution` (REQ-24.1), es distinto de `<directorio>/openspec`: crear ese directorio produciría un segundo planning home y la ambigüedad de REQ-24.6 en todos los verbos. La comparación es entre rutas y NO exige que `<directorio>/openspec` exista, porque `init` es quien lo crea; la existencia del repositorio de especificaciones que exige REQ-24.7 (caso 4) es una condición del primitivo y no de esta comparación. El rechazo DEBE aplicarse con cualquier perfil, finalizar con código de salida distinto de cero, no escribir ni registrar nada y nombrar el puntero, el planning home resuelto y la continuación: inicializar el `workspaceRoot` resuelto con `axiom init --path <workspaceRoot>` o montar la topología a mano según la receta documentada (REQ-24.12).

Cuando el puntero del propio directorio no puede resolverse (por ejemplo, un puntero roto o un `axiom.yaml` apuntado ilegible, REQ-24.7), el primitivo rechaza la resolución y `axiom init` DEBE rechazarse con ese mismo error, que nombra la causa y la continuación, con código de salida distinto de cero y sin escribir ni registrar nada.

`axiom init` NO DEBE rechazarse cuando el puntero resuelve a `<directorio>/openspec` (exista ya ese directorio o no), cuando el propio directorio no contiene puntero (aunque lo contenga un directorio ancestro: el marcador más cercano gana, REQ-24.2) ni cuando no hay ningún marcador: en esos casos se comporta como describen los puntos 1 a 4.

(Previously: creaba siempre `openspec/specs/`, también cuando `axiom.yaml` ya declaraba una baseline del proyecto, y no comprobaba si el directorio contenía un puntero hacia otro planning home ni si ese puntero podía resolverse)

#### Scenario: Inicialización exitosa de proyecto virgen
- **DADO** un directorio limpio sin archivos de Axiom
- **CUANDO** se ejecuta `axiom init --name "MiProyecto"`
- **ENTONCES** se crea `axiom.yaml` con la estructura canónica
- **Y** se crean las carpetas `openspec/specs/`, `openspec/changes/` y `.axiom/inbox/skills/`
- **Y** el proyecto queda registrado en `~/.axiom/workspaces.json`

#### Scenario: Inicialización sobre un `axiom.yaml` que declara una baseline del proyecto
- **DADO** un directorio con un `axiom.yaml` que declara `baseline:` con al menos un documento
- **CUANDO** se ejecuta `axiom init`
- **ENTONCES** no se crea `openspec/specs/` ni `openspec/INDEX.md`
- **Y** se crean `openspec/changes/` y `.axiom/inbox/skills/`
- **Y** la salida informa de que la baseline es del proyecto y el código de salida es `0`
- **Y** el `axiom.yaml` existente no se sobreescribe

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

---

