# Diseño: preflight y sellado derivados de la topología (C-01-b)

## Enfoque técnico

Se implementa el enfoque combinado de la propuesta (decisiones 1-9): **un único contrato de grupos por topología**, con el hook de Claude Code y el plugin de OpenCode como autoridades que leen la topología del directorio de la sesión, y el sellado del kickoff como segunda barrera que aplica la misma topología en la CLI.

La topología efectiva sale siempre de `workspace.Locate` mediante un único método nuevo, `Location.GoverningTopology()`, que fija cómo se trata un `axiom.yaml` sin topología válida: igual que el binario hoy, que conserva la **resolución embebida** cuando un `axiom.yaml` directo no se puede cargar (REQ-24.7, caso 3; `locateFromConfig`, `internal/workspace/locate.go:330-349`). Por tanto, una topología ausente o desconocida se gobierna como `monorepo-embedded`, que es además el conjunto de preguntas más amplio (cierre seguro: se pregunta de más, nunca de menos).

El texto proyectado sigue siendo global y se valida por igualdad exacta; incorpora la orden de leer `workspace.topology`, los grupos de cada topología y marcadores nominales. Los valores que la topología fija (ritmo `interactive` en `multirepo`, almacén `hybrid` siempre) son siempre los conservadores y los construye el runtime, nunca el modelo.

## Decisiones de arquitectura

### D1. Topología efectiva: `Location.GoverningTopology()`

**Elección**: método nuevo en `internal/workspace/locate.go`:

| Situación de `Locate` | Resultado |
|---|---|
| `Config != nil` | `(Config.Workspace.Topology, true)` (la carga ya garantiza `monorepo-embedded` o `multirepo`, `loader.go:150-161`) |
| `ConfigPath != ""` y `Config == nil` (axiom.yaml directo no cargable: topología ausente, desconocida o retirada) | `(TopologyMonorepoEmbedded, true)` — resolución embebida, como hoy |
| `ConfigPath == ""` (sin axiom.yaml: origen `openspec` o `start`) | `("", false)` — no hay topología que aplicar |

Un puntero roto ya hace fallar `Locate` (`ResolveOwnPointer`, `locate.go:308-314`); cada consumidor decide qué hace con ese error (D3, D7).

**Alternativas consideradas**: (a) que cada consumidor lea `loc.Config.Workspace.Topology` y trate `nil` a su manera; (b) un valor por defecto `multirepo`.
**Justificación**: (a) dispersa la regla en tres sitios (hook, sellado, plugin) y ya hay dos lectores del almacén duplicados (`artifact_store.go` y `sddstatus/status.go`) como precedente de deriva. (b) relajaría el cuestionario (sin ritmo) ante un fichero roto. La resolución embebida es lo que el binario ya hace con un `axiom.yaml` no cargable, así que no introduce semántica nueva.

### D2. Modelo de datos de grupos y del bloque derivado (hook)

**Elección**: dos tablas en `internal/cli/sdd_preflight_hook.go`, separadas a propósito:

```go
// sddPreflightGroup es un grupo del cuestionario. ID es el marcador nominal
// estable (`<prefijo><ID>:`); LegacyNumber es su posición en el marcador
// numerado heredado `<prefijo>N/3:` (0 si no tuvo). RequiredIn lista las
// topologías que lo preguntan; un grupo sin topologías solo se reconoce
// para tolerarlo e ignorarlo (respuestas heredadas).
type sddPreflightGroup struct {
	ID           string
	Name         string // nombre legible para los mensajes de rechazo
	LegacyNumber int
	Choices      []sddPreflightChoice
	RequiredIn   []workspace.TopologyType
}

var sddPreflightGroups = []sddPreflightGroup{
	{ID: "pace", Name: "Pace", LegacyNumber: 1,
		Choices:    []sddPreflightChoice{{"Interactive", "interactive"}, {"Automatic", "auto"}},
		RequiredIn: []workspace.TopologyType{workspace.TopologyMonorepoEmbedded}},
	{ID: "artifacts", Name: "Artifacts", LegacyNumber: 2,
		Choices: []sddPreflightChoice{{"OpenSpec", "openspec"}, {"Engram", "engram"}, {"Both", "hybrid"}}},
	{ID: "pr-strategy", Name: "PR strategy", LegacyNumber: 3,
		Choices:    []sddPreflightChoice{{"Ask me", "ask-on-risk"}, {"Single PR", "single-pr"}, {"Auto", "auto-chain"}},
		RequiredIn: []workspace.TopologyType{workspace.TopologyMonorepoEmbedded, workspace.TopologyMultirepo}},
}

// sddPreflightBlockField es una línea `- <Label>: <valor>` del bloque. Su
// valor sale del grupo GroupID cuando la topología lo pregunta; si no, es
// Default. Allowed es el dominio cerrado que valida el bloque construido.
type sddPreflightBlockField struct {
	Label   string
	GroupID string // "" = siempre derivado
	Default string
	Allowed []string
}

var sddPreflightBlockFields = []sddPreflightBlockField{
	{Label: "Pace", GroupID: "pace", Default: "interactive", Allowed: []string{"interactive", "auto"}},
	{Label: "Artifact store", Default: "hybrid", Allowed: []string{"hybrid"}},
	{Label: "Delivery strategy", GroupID: "pr-strategy", Default: "single-pr", Allowed: []string{"ask-on-risk", "single-pr", "auto-chain"}},
}
```

- **Grupos requeridos**: `sddPreflightRequiredGroups(topology)` devuelve, en el orden de la tabla, los grupos cuyo `RequiredIn` contiene la topología. `monorepo-embedded` → `pace`, `pr-strategy`; `multirepo` → `pr-strategy`.
- **Bloque**: cabecera, línea de autoridad, una línea por campo en el orden de `sddPreflightBlockFields`, y `- Review policy: 400 changed lines`. En este cambio sigue teniendo seis líneas y los mismos rótulos, de modo que los consumidores del texto (prompts de las fases, plugin) no cambian de forma.
- **Valor por defecto de `Delivery strategy`**: `single-pr` (P10) cuando el grupo `pr-strategy` no se pregunta. En este cambio el grupo es requerido en las dos topologías, así que el valor por defecto solo se ejercita en la prueba de cuestionario vacío (D5); C-02-c solo tendrá que retirar `pr-strategy` de `RequiredIn` o de la tabla.
- **Extensibilidad**: C-02-c retira el grupo sin tocar los demás; C-06-b añade una fila a `sddPreflightGroups` y, si el resultado debe llegar a las fases, una fila a `sddPreflightBlockFields`. `validSDDPreflightHookBlock` deja de contar seis líneas fijas y valida contra la tabla de campos (`3 + len(fields)` líneas), así que una fila nueva no exige reescribir la validación.

**Alternativas consideradas**: (a) una sola tabla en la que cada grupo lleva su línea de bloque; (b) conservar índices fijos `tokens[0..2]`.
**Justificación**: el almacén demuestra que hay líneas del bloque sin grupo (siempre derivadas) y grupos sin línea (el heredado `artifacts`); mezclarlos en una tabla obliga a casos especiales. Los índices fijos son exactamente el punto rígido que señala la exploración (`sdd_preflight_hook.go:506-513`, `:547-555`).

### D3. Topología en el hook y cálculo de grupos

**Elección**: tras `sddPreflightHookInAxiomProject(payload.CWD)` (que se conserva como puerta de inercia fuera de proyectos Axiom), el hook resuelve la topología con `sddPreflightHookTopology(cwd)`:

```go
func sddPreflightHookTopology(cwd string) workspace.TopologyType {
	loc, err := workspace.Locate(cwd) // cwd vacío → directorio del proceso, como hoy
	if err != nil {
		return workspace.TopologyMonorepoEmbedded
	}
	if topology, ok := loc.GoverningTopology(); ok {
		return topology
	}
	return workspace.TopologyMonorepoEmbedded
}
```

Un error de `Locate` (puntero roto) o un marcador `openspec/` más cercano que el `axiom.yaml` caen en `monorepo-embedded`: el hook nunca falla abierto ni con pánico por la topología. El `cwd` lo envía el runner de Claude Code en el payload; el modelo no puede elegirlo.

**Alternativas consideradas**: sustituir `IsAxiomProject` por `Locate`.
**Justificación**: `IsAxiomProject` excluye a propósito un `openspec/` suelto (`project.go:10-26`); `Locate` lo acepta como marcador. Mantener las dos llamadas conserva la semántica de inercia y deja `Locate` solo para la topología.

### D4. Marcadores nominales, heredados y validación por significado

**Elección**: `sddPreflightQuestionGroup(question) (*sddPreflightGroup, bool)` reconoce, bajo cualquiera de los dos prefijos aceptados (`Gentle AI SDD preflight `, `Axiom SDD preflight `):

- el marcador nominal `<prefijo><ID>:` con un `ID` de `sddPreflightGroups` (comparación exacta, minúsculas);
- el marcador numerado heredado `<prefijo>N/3:` con `N` = `LegacyNumber` (1 → `pace`, 2 → `artifacts`, 3 → `pr-strategy`).

`resolveSDDPreflightHookBlock(questions, answers, topology)`:

1. Si ninguna pregunta lleva prefijo, no reconoce nada (`recognized=false`), como hoy.
2. Asigna cada pregunta a su grupo. Una pregunta con prefijo y marcador desconocido, o de un grupo no requerido en la topología (incluido `artifacts`), **se ignora** sin validar sus opciones ni su respuesta (tolerancia de grupos extra y de texto más nuevo que el binario).
3. Para cada grupo requerido exige exactamente una pregunta (dos preguntas del mismo grupo, sean nominales o heredadas, se rechazan), selección única, oferta del conjunto exacto de etiquetas del grupo (sin duplicados ni ajenas; el orden no importa), una respuesta por el texto de la pregunta y que esa respuesta sea una de las etiquetas (normalización actual: espacios, mayúsculas y sufijo `(Recommended)`).
4. Construye el bloque con D2: el valor de cada campo sale del grupo si la topología lo requiere; si no, de `Default`. En `multirepo`, un ritmo contestado se ignora y el bloque dice `interactive`.

Ya no existe el error «must ask exactly N questions». Mensajes de rechazo (en inglés, como los actuales), con la causa concreta:

| Causa | Mensaje |
|---|---|
| Falta un grupo requerido | `topology <t> requires the SDD preflight group <id> (<Name>), but no question starts with its "Axiom SDD preflight <id>:" marker` |
| Grupo repetido | `SDD preflight group <id> (<Name>) appears in more than one question` |
| Selección múltiple | `SDD preflight question <id> (<Name>) must be single-select` |
| Opción ajena | `SDD preflight question <id> (<Name>) offers option "<x>", which is not one of its labels: <labels>` |
| Opción repetida | `SDD preflight question <id> (<Name>) offers "<x>" more than once` |
| Faltan opciones | `SDD preflight question <id> (<Name>) must offer every one of its labels: <labels>` |
| Sin respuesta | `SDD preflight question <id> (<Name>) has no answer` |
| Respuesta ajena | `SDD preflight question <id> (<Name>) answer "<x>" is not one of its labels: <labels>` |

`sddPreflightHookMissingAuthorityReason(rejection, topology)` sustituye «Ask the canonical grouped preflight» por «Ask the SDD preflight groups that topology `<t>` requires (`<id>`, …) with AskUserQuestion».

**Alternativas consideradas**: (a) marcador `N/M` con `M` variable; (b) rechazar marcadores desconocidos.
**Justificación**: (a) liga el marcador a la posición y al total, que cambian con cada grupo que entra o sale (C-02-c, C-06-b) y con la topología; el nominal es estable. (b) rompería un binario antiguo frente a un texto más nuevo; el hook valida el significado de lo requerido, no la presentación.

### D5. Cuestionario vacío (cero grupos requeridos)

**Elección**: en `runSDDPreflightHook`, el orden pasa a ser:

1. Evento/herramienta/fase SDD (sin cambios) → inercia fuera de Axiom (sin cambios) → id de sesión válido (sin cambios).
2. Denegación de hijo (`agent_id`/`agent_type`) y de bloque escrito por el modelo (`## SDD Session Preflight` en el prompt): **se conservan siempre**, también con cero grupos, porque no dependen de la transcripción y garantizan que el bloque solo lo antepone el hook.
3. `topology := sddPreflightHookTopology(payload.CWD)`; `required := sddPreflightRequiredGroups(topology)`.
4. Si `len(required) == 0`: construye el bloque enteramente derivado (`sddPreflightDerivedBlock(topology)`), lo valida con `validSDDPreflightHookBlock` y permite el despacho **sin exigir** el enlace de `transcript_path` ni consultar la transcripción.
5. Si hay al menos un grupo: enlace de transcripción y `resolveSDDPreflightTranscriptAuthority(path, sessionID, topology)` exactamente como hoy.

La línea de autoridad del bloque no cambia («Parent-confirmed by the runtime; …»): los valores derivados los fija el runtime, no el modelo, y cambiar esa línea rompería la paridad con el plugin y la validación exacta.

**Alternativas consideradas**: exigir igualmente una transcripción ligada sin preguntas.
**Justificación**: no hay ninguna interacción que corroborar; exigir el enlace solo añadiría un fallo sin valor. El riesgo de que el modelo edite `axiom.yaml` para quitarse preguntas queda acotado porque los valores derivados son siempre los conservadores (`interactive`, `hybrid`, `single-pr`) y la edición queda en el historial de Git.

**Prueba sin topología real de cero grupos**: la prueba sustituye `sddPreflightGroups` por una copia sin `multirepo` en `RequiredIn` de `pr-strategy` (con `t.Cleanup`; el fichero de pruebas del hook no usa `t.Parallel`, y la prueba nueva tampoco).

### D6. Texto proyectado global (`session_preflight.go`)

**Elección**: `sddSessionPreflightBody` se reescribe (texto exacto en «Interfaces / Contratos»): orden de leer `workspace.topology` de `axiom.yaml` (ausente o ilegible → grupos de `monorepo-embedded`), grupos por topología, marcadores nominales, «nunca preguntes el almacén: siempre `hybrid`», sin grupo ni mapeos de Artifacts, y la frase de cuestionario vacío. La lista de grupos deja de ser numerada (`- **Pace** (`pace`, …)`), así que el ancla de la variante sin UI pasa de `"\n\n1. **Pace**"` a `"\n\n- **Pace** (`pace`"`.

`sddSessionPreflightBlockWithTool("")` sigue sustituyendo los mismos dos tramos, con textos nuevos: sintaxis de respuesta solo con los grupos requeridos (`Pace: <label>; PR strategy: <label>`), «every required group» en lugar de «all three», y el resumen de mapeos incluye los valores que fija la topología.

`validateSDDSessionPreflightProjection` mantiene la igualdad byte a byte con el bloque canónico; el rechazo específico de `Both -> both` se conserva (sigue dando un mensaje más claro para una instalación muy antigua y no cuesta nada).

**Impacto en `inject.go` y `orchestrator.go`**: ninguno estructural. `renderClaudeSessionPreflight`, la proyección de OpenCode/Kilo, `validateSessionPreflightOverlay` (`inject.go:1052-1087`) y `migratePreservedSDDSessionPreflight` reemplazan el bloque entre sus marcadores, así que `axiom sync` migra las instalaciones existentes sin pasos extra.

**Instalaciones existentes hasta `axiom sync`**: proyectan el texto antiguo (tres preguntas numeradas, con Artifacts). Con el binario nuevo: en `monorepo-embedded` se usan `1/3` y `3/3` y se ignora `2/3`; en `multirepo` se usa `3/3` y se ignoran `1/3` y `2/3`. Ningún preflight antiguo válido se deniega.

**Alternativas consideradas**: proyección distinta por topología.
**Justificación**: el texto se instala una vez por máquina y se valida por igualdad exacta (exploración §1); no puede conocer la topología.

### D7. Plugin de OpenCode: lee la topología

**Elección**: el plugin **sí lee la topología** (tiene `ctx.location.directory` en v2 y `directory` en v1, y ya resuelve `axiom.yaml` y punteros en el bloque `axiom-project-gate`). Se añade un bloque nuevo y delimitado `// axiom-topology:begin … // axiom-topology:end`, separado del bloque de la puerta (que tiene su propia prueba de paridad y no se toca):

- `findAxiomConfig(start)`: misma búsqueda ascendente que `isAxiomProject` (axiom.yaml o puntero que resuelve), devolviendo la ruta.
- `readAxiomTopology(start)`: lee la clave `topology` del mapa de bloque `workspace:` con un lector de líneas (sangría, comillas y comentarios, como `pointerScalar`); `monorepo-embedded` o `multirepo` → ese valor; cualquier otra cosa o fallo de lectura → `monorepo-embedded` (D1).

Contrato replicado del hook: `SDD_PREFLIGHT_GROUPS` con `id`, `legacyNumber`, `requiredIn`, opciones con `token`; `SDD_PREFLIGHT_BLOCK_FIELDS` con `label`, `groupId`, `default`; marcador nominal y numerado heredado; grupos extra o heredados ignorados; enlace etiqueta → token (deja de ser posicional, `sddPreflightBlock` actual `:209`, `:228-230`).

- `canonicalizeSDDPreflightQuestions` deja de exigir tres preguntas y deja de asignar marcadores por posición: solo canonicaliza las preguntas cuyo marcador reconoce de un grupo vigente (`pace`, `pr-strategy`), reescribiendo el marcador heredado al nominal con el prefijo Axiom, forzando selección única y las etiquetas canónicas. Las preguntas no reconocidas o del grupo heredado `artifacts` no se tocan.
- `sddPreflightBlock(args, metadata, topology)`: las respuestas de OpenCode son posicionales respecto a las preguntas (lo impone la herramienta `question`); la opción elegida se enlaza por etiqueta. Exige los grupos requeridos por la topología con los mismos motivos de rechazo que el hook.
- `execute.before` (subagente): si no hay preflight guardado y la topología no requiere ningún grupo, antepone el bloque derivado; se conservan la denegación de sesión hija y de bloque escrito por el modelo.
- Cada copia conserva su normalización de respuestas actual (v1 con NFD, v2 con recorte y minúsculas): fuera de alcance.
- **Digest**: antes de editar, se añade en `opencode_plugin_digests.go` el sha256 de la revisión vigente de cada copia (`git show HEAD:internal/assets/opencode/plugins/sdd-task-result-artifacts.ts | sha256sum` → lista `"v1"`; ídem `plugins-v2/` → lista `"v2"`), con el commit corto en el comentario.

**Alternativas consideradas**: aceptar cualquier subconjunto no vacío de grupos conocidos sin conocer la topología.
**Justificación**: sin topología el plugin no puede detectar el cuestionario vacío que C-02-c necesitará en `multirepo` (denegaría «preflight missing») ni exigir el ritmo en `monorepo-embedded`. El riesgo de divergencia del lector de líneas frente a `yaml.v3` (estilo de flujo `workspace: {topology: …}`) acaba en `monorepo-embedded`, que es cierre seguro, y lo fija una prueba de paridad con `Locate` (pruebas, U5). Los `axiom.yaml` que genera Axiom (`buildAxiomYamlWithRoles`) usan estilo de bloque.

### D8. Sellado: regla pura en `kickoff`, invocada por la CLI

**Elección**:

1. `ParseSealArgs` sigue pura y **deja de rechazar** la ausencia de `--execution-style` y `--from-session-pace` (deja `ExecutionStyle` vacío) y **deja de aplicar** `checkContinuousRoster`. El resto de su validación sintáctica no cambia (incluido `--docs-publication` solo con `continuous`).
2. Método puro nuevo en `internal/kickoff/args.go`:

```go
// ApplyTopology resuelve el estilo y valida el roster del sellado explícito
// para la topología que gobierna el cambio ("" sin axiom.yaml). Orden:
// reglas de topología (con mensaje que nombra la topología), estilo
// obligatorio (REQ-21.5) y, como red de seguridad, la regla de continuous
// (REQ-28.12, REQ-28.13). --infer no pasa por aquí.
func (a *SealArgs) ApplyTopology(topology workspace.TopologyType) error

var (
	ErrTopologyForbidsContinuous = errors.New("la topología multirepo sella siempre checkpointed")
	ErrTopologyRequiresFullstack = errors.New("la topología monorepo-embedded solo admite el rol fullstack")
)
```

   - `multirepo`: estilo `continuous` (explícito, derivado de `--from-session-pace auto`, o explícito sobre el ritmo) → error envuelto en `ErrTopologyForbidsContinuous`: «kickoff seal: la topología `multirepo` sella siempre `checkpointed`: `<origen>` no se admite; quita la bandera o sella con `--execution-style checkpointed`». Estilo vacío → `checkpointed` con `ExecutionStyleSource = "topology"`.
   - `monorepo-embedded`: cualquier rol distinto de `fullstack` → error envuelto en `ErrTopologyRequiresFullstack`: «kickoff seal: la topología `monorepo-embedded` solo admite el rol `fullstack` (roles recibidos: core, web); quita las banderas `--role` para sellar con fullstack, o declara `workspace.topology: multirepo` en axiom.yaml si el trabajo necesita roles especializados». La política de compuerta de `fullstack` no se comprueba.
   - Después, para cualquier topología: estilo vacío → el error actual de REQ-21.5, ampliado con «solo `multirepo` deriva `checkpointed`»; y `checkContinuousRoster` (sin cambios de texto).
3. En `runSDDKickoffSeal` (`sdd_kickoff.go`), dentro de la rama sin `--infer` y **antes** de `ToKickoff`, de la comprobación de `SDDErr` y de `applySealBranchPolicy`:

```go
topology, _ := loc.GoverningTopology()
if err := parsed.ApplyTopology(topology); err != nil {
	return err
}
```

   Así un rechazo por topología no emite avisos de rama ni comprueba la rama del cambio, y su mensaje gana a la regla #249 cuando ambas aplican. `ValidateForSeal` (`sdd_kickoff.go:95`) sigue siendo la defensa en profundidad.

Sin `axiom.yaml` (`topology == ""`) no hay regla de topología: el sellado se comporta como hoy (estilo obligatorio y regla #249). Con un `axiom.yaml` no cargable, rige `monorepo-embedded` (D1).

**Alternativas consideradas**: (a) cambiar la firma de `ParseSealArgs` para recibir la topología; (b) dejar la regla #249 en `ParseSealArgs` y añadir la de topología después.
**Justificación**: (a) rompe la pureza que exige la propuesta. (b) haría que en `monorepo-embedded` con `--execution-style continuous --role core` ganara el mensaje #249 en lugar del que nombra la topología, contra la precedencia fijada por la propuesta. Una función pura en `kickoff` conserva la prueba unitaria sin Git ni disco y deja la llamada en la CLI, como pide la decisión 5.

`ExecutionStyleSource` no tiene validación de valores en `kickoff` (solo se escribe); `"topology"` es un valor nuevo informativo.

### D9. Validador del workspace

**Elección** (`internal/workspace/validator.go`), ninguna regla entra en `Locate` ni en `parseLayered`:

- **Paso 3b de `Validate`**, tras el paso 3: con `Topology == monorepo-embedded`, si `cfg.Roles` contiene alguna clave distinta de `fullstack` → un error (claves ordenadas): «la topología 'monorepo-embedded' solo admite el rol 'fullstack', pero axiom.yaml declara: core, qa. Para migrar, deja en 'roles' solo 'fullstack' (con los repositorios y tecnologías de los demás) o pasa a 'workspace.topology: multirepo' con un 'specs_repository' propio; los kickoff.yaml ya sellados se siguen leyendo igual». Lo heredan `axiom workspace validate`, el dashboard (`service.go:172`, el proyecto aparece no conforme con ese mensaje) y `ValidateInit` del hub. El doctor (`doctor_workspace.go:66-69`) borra los roles antes de validar y no lo reporta; se deja así (fuera de alcance).
- **`storeErrors`**: además del error actual por `technical_context.maintained`, un error incondicional cuando `openspec/config.yaml` declara `engram` o `none`: «openspec/config.yaml declara artifact_store '<s>', que Axiom ya no admite: el almacén de los cambios es siempre 'hybrid', y con '<s>' no hay paradas de tres salidas, publicación de compuertas ni relevo (--hand-off). Para migrar, quita la clave artifact_store o declara 'artifact_store: hybrid'». `openspec`, `hybrid` o sin declarar no producen error. `DeclaredArtifactStore` y `sddstatus` no cambian (el valor por defecto sigue siendo `openspec`).

**Alternativas consideradas**: rechazar en la carga.
**Justificación**: la propuesta exige que `Locate` siga cargando ambos casos; romper la carga dejaría inservibles status, archivado y lectura de `kickoff.yaml` existentes.

### D10. Alta del proyecto (hub y dashboard)

**Elección**:

- `planConfig` (`internal/hub/init_validate.go:174-205`): el perfil `knowledge`/`spec-only` sigue forzando `multirepo` con el rol `knowledge` (primera rama, sin cambios). Los roles **detectados** (`tech.ConfiguredRoles`) solo se aplican si la topología normalizada es `multirepo`; en `monorepo-embedded` se cae al rol único `fullstack` actual. Los roles **explícitos** (`opts.Roles`, CLI o petición del dashboard) se conservan tal cual y, en `monorepo-embedded` con claves distintas de `fullstack`, los rechaza `ValidateInit` mediante D9 antes de escribir nada (no se descartan en silencio).
- `buildAxiomYaml` (`internal/hub/init.go:355-388`): la misma regla para los roles detectados.
- `detector.go` no cambia: sigue detectando roles; quien decide aplicarlos es el plan según la topología.
- Dashboard: `projects.js` oculta `#roles-builder-container` cuando `#select-project-topology` vale `monorepo-embedded` (y lo muestra en `multirepo`); `validateProjectForm` devuelve `roles: []` sin violaciones de roles en `monorepo-embedded`, de modo que el servidor escribe el `fullstack` canónico (`<nombre> Fullstack`, tecnología detectada). `buildRoles` no cambia.

**Alternativas consideradas**: que `buildRoles` devuelva un `fullstack` construido en JavaScript; hacer topológico el detector.
**Justificación**: el servidor ya construye el `fullstack` canónico con su nombre y tecnologías; duplicarlo en JS crea dos fuentes. El detector se usa también para recomendar tecnologías y su salida es útil en `multirepo`.

### D11. Doctrina del almacén y del ritmo

**Elección**: la doctrina deja explícito que:

- El almacén **nunca se pregunta**; el bloque de sesión dice siempre `hybrid` (política de sesión).
- Cada lanzamiento de fase sigue pasando como `artifact_store.mode` el `artifactStore` (y `artifactPaths`) que devuelve `axiom sdd status` — autoridad nativa —, que vale `openspec` cuando `openspec/config.yaml` no declara nada. El `hybrid` de la sesión **nunca lo sustituye**. No se cambia el valor por defecto de `sddstatus`.
- El ritmo depende de la topología: `multirepo` siempre Interactive sin preguntar; `monorepo-embedded` se pregunta y no hay valor por defecto («default to Automatic» desaparece).
- El sellado: en `multirepo` se sella sin `--execution-style` (o con `checkpointed`/`--from-session-pace interactive`); en `monorepo-embedded`, solo `fullstack`.

Textos y ficheros concretos en «Interfaces / Contratos» y «Cambios por fichero». Las secciones de las paradas de tres salidas («con `engram` o `none` la salida 2 no está disponible») no se tocan: siguen siendo ciertas para declaraciones heredadas que el status todavía resuelve.

**Alternativas consideradas**: que el orquestador pase `hybrid` a las fases.
**Justificación**: la regla B de `sdd-phase-common.md` prohíbe a las fases derivar el almacén, y el binario (publicación, relevo, contexto técnico) trata `openspec` ≡ `hybrid` (exploración §5); pasar un valor distinto del status desalinearía las fases con su autoridad.

## Flujo de datos

```
                 axiom.yaml (workspace.topology)
                          │
                 workspace.Locate(cwd)
                          │
                Location.GoverningTopology()  ── nil/no cargable → monorepo-embedded
          ┌───────────────┼──────────────────────────┐
          │               │                          │
   Hook PreToolUse   kickoff seal (CLI)      Plugin OpenCode (TS,
   (cwd del runner)  ApplyTopology(topo)     lector de líneas espejo)
          │               │                          │
  grupos requeridos   estilo/roles por           grupos requeridos
  (tabla de grupos)   topología → #249 red       (tabla espejo)
          │               │                          │
  transcripción ──→  kickoff.yaml             respuestas `question`
  (≥1 grupo)                                         │
  o derivado (0)                                      │
          └──────────→ bloque `## SDD Session Preflight` ←──┘
                       (campos: Pace, Artifact store=hybrid,
                        Delivery strategy, Review policy)
                                  │
                       prompt de la fase SDD
       (la fase persiste en el artifactStore de `axiom sdd status`)
```

## Cambios por fichero

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/workspace/locate.go` | Modificar | `Location.GoverningTopology()` (D1) |
| `internal/workspace/validator.go` | Modificar | Paso 3b monorepo solo `fullstack`; `storeErrors` incondicional para `engram`/`none` (D9) |
| `internal/workspace/locate_test.go` (o fichero nuevo `governing_topology_test.go`) | Modificar/Crear | Pruebas de D1 |
| `internal/workspace/validator_test.go` | Modificar | Casos `:468-522` y nuevos de roles |
| `internal/kickoff/args.go` | Modificar | `ParseSealArgs` sin estilo obligatorio ni roster; `ApplyTopology`; errores centinela; comentario de `ExecutionStyleSource` |
| `internal/kickoff/args_test.go` | Modificar | `TestParseSealArgsRequiresExecutionStyleOrSessionPace`, `TestParseSealArgsContinuousRosterRule` vía `ApplyTopology("")`; pruebas nuevas |
| `internal/cli/sdd_kickoff.go` | Modificar | Llamada a `ApplyTopology` (D8); ayuda (`:38-39`) con la regla de topología |
| `internal/cli/sdd_kickoff_test.go`, `sdd_kickoff_branch_test.go`, `sdd_writer_guard_test.go` | Modificar | Pruebas nuevas de sellado; migración de los sellados monorepo con roles (ver Estrategia de pruebas) |
| `internal/hub/init_validate.go`, `internal/hub/init.go` | Modificar | Roles detectados solo en `multirepo` (D10) |
| `internal/hub/init_validate_test.go`, `init_test.go`, `hub_test.go`, `init_multirepo_test.go` | Modificar | Expectativas de roles por topología |
| `internal/dashboard/assets/js/core/projectform.js`, `internal/dashboard/assets/js/screens/projects.js` | Modificar | Formulario sin roles en monorepo (D10) |
| `internal/dashboard/assets_projectform_test.go` | Modificar | Caso monorepo → `roles: []` |
| `internal/cli/sdd_preflight_hook.go` | Modificar | D2-D5 |
| `internal/cli/sdd_preflight_hook_test.go` | Modificar | Adaptación y pruebas nuevas |
| `internal/assets/opencode/plugins-v2/sdd-task-result-artifacts.ts`, `internal/assets/opencode/plugins/sdd-task-result-artifacts.ts` | Modificar | D7 |
| `internal/components/sdd/opencode_plugin_digests.go` | Modificar | Digest de la revisión sustituida de cada copia |
| `internal/assets/opencode_v2_sdd_test.go`, `internal/assets/opencode_sdd_project_gate_test.go`, `internal/sddtaskresult/preflight_question_harness_test.go`, `transport_parity_test.go`, `internal/components/sdd/opencode_runtime_test.go`, `sdd_task_result_transport_test.go` | Modificar | Marcadores nominales, número variable, enlace por etiqueta |
| `internal/sddtaskresult/topology_reader_parity_test.go` | Crear | Paridad del lector TS de topología con `GoverningTopology` |
| `internal/components/sdd/session_preflight.go` | Modificar | Cuerpo, variante sin UI y ancla (D6) |
| `internal/components/sdd/orchestrator.go` | Modificar | Reescrituras de reserva (`:148-223`) y nuevas cotas de «Execution Mode» |
| `internal/assets/claude/sdd-orchestrator-workflow.md` | Modificar | «Execution Mode» (`:68-73`) y «Artifact Store Mode» (`:97-101`) |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | Kickoff (`:80`), «Execution Mode» (`:255-261`), «Artifact Store Mode» (`:303-313`) |
| `internal/assets/{codex,antigravity,windsurf,cursor,kiro,kimi,opencode,hermes}/sdd-orchestrator.md` | Modificar | Mismas dos secciones; `generic`, `gemini`, `qwen` solo si contienen el texto (los dos últimos resuelven la sección compartida) |
| `internal/assets/skills/_shared/persistence-contract.md` | Modificar | «Mode Resolution» (`:3-9`) |
| `internal/components/sdd/session_preflight_test.go`, `orchestrator_composition_test.go`, `renderer_invariants_test.go`, `inject_test.go`, `orchestrator_shared_sections_test.go`, `persistence_recovery_test.go`, `delivery_strategy_vocabulary_test.go`, `research_state_contract_test.go`, `review_ledger_contract_test.go`, `internal/assets/assets_test.go`, `internal/assets/independent_review_boundary_test.go` | Modificar | Literales del texto y de la doctrina que cambien |
| `testdata/golden/*` (los 11 de la exploración más `sdd-claude-claudemd.golden` y `combined-claude-claudemd.golden` por el párrafo del kickoff) | Regenerar | `go test ./internal/components/ -run TestGolden -update` |

## Interfaces / Contratos

### Cuerpo canónico del cuestionario (`sddSessionPreflightBody`)

```text
### SDD Session Preflight (HARD GATE)

Before every SDD command or affirmative natural-language SDD request, run this preflight before the SDD init guard; cache choices for the session only through runtime-confirmed parent authority. Phrase examples are routing hints, never the authority boundary.

The project topology decides which groups to ask. Before asking, read `workspace.topology` from the project's `axiom.yaml`; when it is missing or unreadable, ask the `monorepo-embedded` groups. `monorepo-embedded` asks Pace and PR strategy. `multirepo` asks PR strategy only: its pace is always Interactive and is never asked. When the topology requires no group, ask nothing: the runtime derives the whole session preflight at SDD dispatch. Never ask the artifact store: it is always `hybrid`.

Always collect this preflight with the `question` tool; never collect these answers as typed chat text and never fall back to a plain-chat prompt while the `question` tool exists. If the runtime rejects the grouped result, fix the reported problem and ask again with the `question` tool.
Ask every group the topology requires in ONE `question` tool call; no sequential wizard and no separate calls. Each native question text must start with the exact host marker of its group: `Axiom SDD preflight pace:` or `Axiom SDD preflight pr-strategy:`. The marker is runtime metadata; keep the option labels below byte-exact so the runtime can bind their semantics, while localizing only the remaining question text and descriptions to the conversation language and persona. Option order does not matter: the runtime binds every answer by its label.

- **Pace** (`pace`, `monorepo-embedded` only): Interactive or Automatic.
- **PR strategy** (`pr-strategy`): Ask me, Single PR, or Auto.

Only a successful parent `question` result with one offered answer per required group establishes native preflight authority. Model-authored defaults, summaries, headings, installed assets, prior sessions, and child-agent prose do not. The runtime derives and prepends the canonical `## SDD Session Preflight` block at SDD dispatch, including the values the topology fixes (`multirepo` pace `interactive`, artifact store `hybrid`); never write or duplicate that block yourself. Missing authority blocks dispatch and requires the parent to ask the required groups.

Review policy is fixed at 400 changed lines per PR; above 400, split the PR or require maintainer-approved `size:exception`; NEVER ask it as another group or selectable budget.

Canonical mappings:
- Interactive -> `interactive`
- Automatic -> `auto`
- Ask me -> `ask-on-risk`
- Single PR -> `single-pr`
- Auto -> `auto-chain`
```

### Variante sin UI nativa (`tool == ""`)

- Tramo 1 (desde «Always collect this preflight with the `question` tool» hasta `"\n\n- **Pace** (`pace`"`):
  «This runtime has no classified native question UI. Present ONE complete blocking prompt in plain chat or terminal with every group the topology requires, in the order below, and explain that their answers are required before init. Each group is single-select; its listed labels are the complete allowed-answer domain. Include every label and description, the fixed review policy, and this answer syntax with only the required groups: `Pace: <label>; PR strategy: <label>`. Then STOP; never default, infer, or launch dependent work. Validate every explicit parent answer under the Lossless Blocking Prompts contract before caching its canonical mapping.»
- Tramo 2 (párrafo de autoridad):
  «Only explicit, validated parent answers to every required group establish preflight authority in this fallback runtime. Model-authored defaults, summaries, headings, installed assets, prior sessions, and child-agent prose do not. After validating every answer, summarize the canonical mappings once, together with the values the topology fixes (`multirepo` pace `interactive`, artifact store `hybrid`), and pass them to each SDD phase; missing authority blocks dispatch and requires the parent to re-present the complete prompt. When the topology requires no group, pass only the fixed values.»

### Doctrina («Execution Mode», «Artifact Store Mode»)

- Frase que sustituye a «If the user doesn't specify, default to **Automatic**.» en todos los assets: «No mode is assumed by default: `multirepo` always runs **Interactive** and never asks; `monorepo-embedded` asks it in the session preflight.» `composeFallbackSessionPreflight` (`orchestrator.go:170-180`) cambia su cota de inicio a esa frase exacta en el mismo commit (si no, entra en pánico y lo detectan los goldens).
- La apertura de «Execution Mode» de `sdd-orchestrator-sections.md` conserva el prefijo «When the user invokes `/sdd-new`» (cota de `orchestrator.go:171`) y cambia «ASK which execution mode they prefer» por «resolve the execution mode from the project topology (`workspace.topology` in `axiom.yaml`): `multirepo` always uses **Interactive** without asking; in `monorepo-embedded` ASK which mode they prefer».
- «Artifact Store Mode» (cuerpo común, adaptado al estilo de cada asset): «The artifact store is never asked: in Axiom the session store is always `hybrid`. Pass, as `artifact_store.mode` to every phase launch, the `artifactStore` and `artifactPaths` that `axiom sdd status` reports for the change; that native value (`openspec` when `openspec/config.yaml` declares nothing) is the authority phases persist to, and the session `hybrid` never overrides it. `axiom workspace validate` rejects a declared `engram` or `none`.»
- Reescrituras de reserva (`orchestrator.go`): «### Artifact Store Policy» → «The session artifact store is always `hybrid` and is never asked. Pass the `artifactStore` that `axiom sdd status` reports to every phase; never detect or default a separate choice.»; «### Artifact Store Mode» → «Pass the `artifactStore` that `axiom sdd status` reports as `artifact_store.mode` to every sub-agent launch; the session `hybrid` never overrides it.»; Copilot → «Artifact store: use the `artifactStore` that `axiom sdd status` reports.»; Codex → encabezado «### Artifact store (status-reported backend)»; búsqueda de init → «1. Use the `artifactStore` that `axiom sdd status` reports. …» y «… never silently change the reported store.» (se conservan las ramas `openspec`/`engram`/`hybrid` para declaraciones heredadas).
- `persistence-contract.md`, «Mode Resolution»: «The orchestrator passes `artifact_store.mode` with the `artifactStore` that `axiom sdd status` reports (`openspec` or `hybrid` in Axiom; `engram` and `none` only for a legacy declaration, which `axiom workspace validate` rejects). The orchestrator never asks the user for a mode: the session preflight always records `hybrid`, and it never overrides the reported store. If no mode is passed, STOP and report it; never detect or default one.» Las tablas de modos se conservan (las fases siguen ramificando por los cuatro valores).
- Kickoff (`sdd-orchestrator-sections.md:80`): se conservan las frases actuales (la de `continuous` la fija `gatesContinuousFullstackPhrase`) y se añade: «The topology constrains the seal: in `multirepo`, seal without `--execution-style` (the command derives `checkpointed`) or with `checkpointed`; it rejects `continuous` and `--from-session-pace auto`. In `monorepo-embedded`, seal only `fullstack`: any other `--role` is rejected, because specialized roles exist only in `multirepo`.»

### Firmas Go nuevas o cambiadas

```go
// internal/workspace
func (l Location) GoverningTopology() (TopologyType, bool)

// internal/kickoff
func (a *SealArgs) ApplyTopology(topology workspace.TopologyType) error
var ErrTopologyForbidsContinuous, ErrTopologyRequiresFullstack error

// internal/cli (no exportadas)
func sddPreflightHookTopology(cwd string) workspace.TopologyType
func sddPreflightRequiredGroups(topology workspace.TopologyType) []*sddPreflightGroup
func sddPreflightQuestionGroup(question string) (*sddPreflightGroup, bool)
func resolveSDDPreflightHookBlock(questions []sddPreflightHookQuestion, answersRaw json.RawMessage, topology workspace.TopologyType) (string, bool, error)
func resolveSDDPreflightTranscriptAuthority(path, sessionID string, topology workspace.TopologyType) (string, bool, error)
func sddPreflightDerivedBlock(topology workspace.TopologyType) string
func sddPreflightBuildBlock(values map[string]string) string // por GroupID; ausentes → Default
func validSDDPreflightHookBlock(block string) bool            // contra sddPreflightBlockFields
func sddPreflightHookMissingAuthorityReason(rejection error, topology workspace.TopologyType) string
```

## Estrategia de pruebas

Sin binarios sueltos (`axiom.exe`): solo `go test` por paquete y con `-run` en `internal/cli` (el paquete entero no cabe). TDD: RED observado antes de cada unidad.

| Capa | Qué | Cómo |
|---|---|---|
| Unidad `workspace` | `GoverningTopology` (cargado, no cargable, sin axiom.yaml); `Validate` monorepo con `core`/`qa` → error con migración; multirepo con roles → sin error; `ValidateDeclarationPaths` con `engram`/`none` sin contexto mantenido → error; `Locate` sigue cargando ambos | `TestLocationGoverningTopology`, `TestValidateMonorepoRejectsSpecializedRoles`, `TestValidateMultirepoKeepsSpecializedRoles`, `TestValidateDeclarationPathsRejectsLegacyArtifactStore`, `TestLocateLoadsMonorepoRolesAndLegacyStore`; adaptar `validator_test.go:468-522`. `go test ./internal/workspace/` |
| Unidad `kickoff` | `ApplyTopology`: multirepo deriva `checkpointed` (source `topology`), rechaza `continuous` explícito, por ritmo `auto` y explícito sobre ritmo; monorepo rechaza roles (también con `continuous`: gana `ErrTopologyRequiresFullstack`); sin topología conserva REQ-21.5 y #249 | `TestSealArgsApplyTopologyMultirepo`, `TestSealArgsApplyTopologyMonorepo`, `TestSealArgsApplyTopologyPrecedence`, `TestSealArgsApplyTopologyWithoutTopology`; adaptar `TestParseSealArgsRequiresExecutionStyleOrSessionPace`, `TestParseSealArgsContinuousRosterRule`. `go test ./internal/kickoff/` |
| Integración CLI (seal) | De extremo a extremo con fixtures reales; un rechazo no escribe `kickoff.yaml` ni emite avisos de rama; `--infer` exento | `TestRunSDDKickoffSealMultirepoDerivesCheckpointed`, `TestRunSDDKickoffSealMultirepoRejectsContinuous`, `TestRunSDDKickoffSealMonorepoRejectsRolesBeforeBranchGuard`, `TestRunSDDKickoffSealInferIsExemptFromTopology`. `go test ./internal/cli/ -run 'TestRunSDDKickoff|TestKickoffSeal'` |
| Migración de pruebas CLI | Los sellados de monorepo con `core`/`qa` (`sdd_kickoff_branch_test.go:156,242,293`; `sdd_writer_guard_test.go` vía `guardProjectSealedWith`) pasan por un ayudante de prueba `sealLegacyKickoff(t, ws, args...)` que replica `runSDDKickoffSeal` sin `ApplyTopology` (`ParseSealArgs` → `ToKickoff` → `applySealBranchPolicy` → `kickoff.Seal`) y documenta que simula un kickoff sellado antes de C-01-b; `TestKickoffSealAcceptsTheRoleBranchesOfTheRolesBeingSealed` pasa a afirmar el rechazo por topología. Los sellados sin `axiom.yaml` (`sdd_gate*_test.go`, `newGovernanceWorkspace`) y los de multirepo no cambian | `go test ./internal/cli/ -run 'TestKickoffSeal|TestWriter|TestGate|TestRunSDDGate'` |
| Unidad hook | Grupos por topología; nominales; heredados con Artifacts ignorado; multirepo con solo `pr-strategy` y ritmo contestado ignorado; falta de grupo requerido con causa que nombra id y topología; grupo duplicado nominal+heredado; marcador desconocido ignorado; bloque siempre `hybrid`; cero grupos (tabla sustituida) permite sin transcripción y sigue denegando hijo y bloque del modelo; `Locate` con error → grupos de monorepo | `TestSDDPreflightHookRequiredGroupsByTopology`, `TestSDDPreflightHookAcceptsNominalMarkers`, `TestSDDPreflightHookAcceptsLegacyNumberedMarkersAndIgnoresArtifacts`, `TestSDDPreflightHookMultirepoAsksOnlyPRStrategy`, `TestSDDPreflightHookMultirepoIgnoresAnsweredPace`, `TestSDDPreflightHookDeniesMissingRequiredGroup`, `TestSDDPreflightHookDeniesGroupAskedTwice`, `TestSDDPreflightHookIgnoresUnknownNamedGroup`, `TestSDDPreflightHookZeroRequiredGroupsAllowsDerivedBlock`, `TestSDDPreflightHookZeroRequiredGroupsKeepsChildAndModelDenials`, `TestValidSDDPreflightHookBlockFollowsFieldTable`; adaptar `…AllowsCorroboratedCanonicalPreflight` (espera `hybrid`), `…RejectsMalformedMarkerUnderEveryPrefix`, `…BindsAnswersByLabelNotPosition`, `…DenialNamesTheCause`, `…DeniesOutsideDomainAnswer`, `…MultiSelectAnswer`. Payload con `cwd` explícito (`workspacetest.Embedded`, `workspacetest.Canonical`); los payloads sin `cwd` siguen resolviendo el `axiom.yaml` del propio repositorio (monorepo). `go test ./internal/cli/ -run 'SDDPreflight'` |
| Plugin OpenCode (bun) | Ambas copias: monorepo con nominales; multirepo con solo `pr-strategy`; heredados `1/3`-`3/3`; opciones reordenadas enlazadas por etiqueta; falta de grupo requerido; cero grupos (fixture de tabla) antepone el bloque derivado; sin asignación posicional de preguntas sin marcador; lector de topología con paridad frente a `GoverningTopology` (bloque, comillas, comentarios, puntero, valor desconocido, sin axiom.yaml) | Adaptar `TestOpenCodeV2SDDDispatch`, `TestOpenCodeV2SDDPreflightAcceptsBothPrefixes`, `TestOpenCodePreflightQuestionCanonicalizationAndTolerantMatching` (escenario A pierde la asignación posicional), `transport_parity_test.go`, `opencode_runtime_test.go`, `sdd_task_result_transport_test.go`; crear `TestOpenCodeTopologyReaderParity`. `go test ./internal/sddtaskresult/ ./internal/assets/ -run 'OpenCode'` y `go test ./internal/components/sdd/ -run 'OpenCode|Digest'` |
| Texto y doctrina | Cuerpo y variante sin UI; validación exacta; reescrituras de reserva con las cotas nuevas; goldens | `TestSessionPreflightBodyNamesTopologyGroups`, `TestSessionPreflightFallbackAnswerSyntaxUsesRequiredGroups`; adaptar los literales de `session_preflight_test.go:25-264` y los tests de doctrina listados. `go test ./internal/components/sdd/` y `go test ./internal/components/ -run TestGolden -update` + revisión del diff + `go test ./internal/components/ -run TestGolden` |
| Hub y dashboard | Monorepo ignora roles detectados; multirepo los aplica; roles explícitos en monorepo rechazados por `ValidateInit`; perfil `knowledge` fuerza multirepo + `knowledge`; formulario monorepo → `roles: []` | `TestPlanConfigMonorepoIgnoresDetectedRoles`, `TestPlanConfigMultirepoAppliesDetectedRoles`, `TestValidateInitRejectsExplicitRolesInMonorepo`, `TestPlanConfigKnowledgeProfileForcesMultirepoKnowledge`; caso nuevo en `TestProjectFormValidationRuntime`. `go test ./internal/hub/ ./internal/dashboard/` |

## Unidades de trabajo y estimación

Orden de commits (cada uno verde con sus pruebas; una sola PR con `size:exception`, coherente con la estrategia de entrega de la sesión):

| # | Unidad | Depende de | Líneas de autor (aprox.) |
|---|---|---|---|
| U1 | `workspace`: `GoverningTopology` y reglas del validador | — | 180 |
| U2 | `kickoff` + CLI de sellado: `ApplyTopology`, llamada en `runSDDKickoffSeal`, ayuda y migración de pruebas CLI | U1 | 370 |
| U3 | Hub y dashboard: alta monorepo solo `fullstack` | U1 | 130 |
| U4 | Hook: tabla de grupos y campos, marcadores, topología, cuestionario vacío | U1 | 600 |
| U5 | Plugin de OpenCode (dos copias), digests y pruebas de bun | U4 (contrato) | 460 |
| U6 | Texto proyectado, reescrituras de `orchestrator.go`, doctrina de todos los runtimes, `persistence-contract.md` y goldens regenerados | U4 | 330 (goldens generados excluidos) |

Total aproximado: 2.070 líneas de autor, más los goldens regenerados. U4 precede a U6 para que ningún commit proyecte marcadores nominales que el hook aún no acepta.

## Matriz de amenazas

El cambio toca una integración de proceso (el hook `PreToolUse(Agent)` que decide el despacho de fases) y un plugin que lee ficheros; no ejecuta comandos, subprocesos, Git ni PR.

| Frontera | Casos adversos mínimos | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas tipo documentación | `requirements.txt`, `README.sh`, Markdown ejecutable | N/A: no se clasifica ni ejecuta ningún fichero | — | — |
| Selección de repositorio | `cwd` relativo, absoluto, ausente | Aplicable: el hook y el plugin eligen el `axiom.yaml` por el directorio de la sesión | El `cwd` sale solo del payload del runner (Claude) o de `ctx.location.directory` (OpenCode); vacío → directorio del proceso (como hoy); relativo → `normalizeStart` lo absolutiza; error de `Locate` o puntero roto → `monorepo-embedded` (más grupos, nunca menos) | `TestSDDPreflightHookRequiredGroupsByTopology` (cwd multirepo, cwd monorepo, sin cwd, puntero roto), `TestOpenCodeTopologyReaderParity` |
| Estado de commit | staged, `commit -a`, índice vacío | N/A: ni el hook ni el plugin ni el sellado hacen commits | — | — |
| Estado de push | rama de seguimiento, primer push, refspec | N/A: sin push | — | — |
| Comandos de PR | `--head`, prefijo de entorno, comandos compuestos | N/A: sin automatización de PR | — | — |
| Autoridad de despacho (específica de este cambio) | despacho de hijo, bloque escrito por el modelo, transcripción no ligada, cuestionario vacío | Aplicable | Con ≥1 grupo, todas las protecciones actuales; con 0 grupos, se conservan la denegación de hijo y de bloque del modelo y solo se omite la transcripción; los valores derivados son siempre los conservadores | `TestSDDPreflightHookZeroRequiredGroupsKeepsChildAndModelDenials` y las pruebas actuales de hijo, bloque del modelo y transcripción no ligada, que siguen en rojo→verde con grupos requeridos |

## Migración / despliegue

- Sin migración de datos: no se reescribe ningún `axiom.yaml`, `openspec/config.yaml` ni `kickoff.yaml`. Los `kickoff.yaml` existentes de monorepo con roles se siguen leyendo y gobernando igual (ramas de rol, guarda de escritores, compuertas por rol).
- Proyectos afectados: un monorepo con roles distintos de `fullstack` o un `artifact_store` `engram`/`none` declarado pasa a fallar `axiom workspace validate` (y el dashboard lo marca no conforme) con un mensaje de migración; su carga, status y archivado no cambian. Este repositorio ya declara solo `fullstack` y no declara `artifact_store`.
- Transición del texto: hasta `axiom sync`, hook y plugin aceptan los marcadores numerados heredados e ignoran la respuesta de almacén.
- Plugin: el digest de la revisión sustituida permite que `axiom sync` reemplace las copias instaladas no modificadas.
- Reversión: revertir la PR y ejecutar `axiom sync`.

## Preguntas abiertas

Ninguna bloqueante. Supuestos a validar en la fase de spec, que corre en paralelo:

- [ ] Los escenarios de spec deben fijar la precedencia de D8 (con `monorepo-embedded` + `continuous` + `core`, el mensaje es el de topología) y el tratamiento de una topología no cargable como `monorepo-embedded` (D1).
- [ ] El doctor (`doctor_workspace.go`) no reporta la regla de roles de monorepo porque valida sin roles; se deja fuera de alcance salvo que la spec lo exija.
