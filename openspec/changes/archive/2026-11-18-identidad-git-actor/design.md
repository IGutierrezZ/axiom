# Diseño: identidad git como actor de los artefactos versionados (C-07-a)

## Enfoque técnico

Se implementa el enfoque A de la propuesta: un **paquete hoja nuevo, `internal/gitidentity`**, que lee `user.name` y `user.email` de la configuración de git con `gitcommon.Run` y expone una identidad saneada; y la **decisión en el adaptador CLI** (`internal/cli`), que es el único sitio que sabe si hay `--actor` explícito, si el sellado es `--infer` y dónde está la raíz del workspace.

El dominio no cambia de forma: `internal/kickoff` sigue sin ejecutar git, `SealArgs.ToKickoff` sigue siendo pura y los tipos de `GateRecord.Actor` y `Kickoff.SealedBy` siguen siendo `string` sin validación de valor. El adaptador hace en `seal` lo mismo que ya hace con `Change` (`internal/cli/sdd_kickoff.go:99`): completa un campo del documento ensamblado antes de escribirlo.

La resolución nunca falla y nunca bloquea: sin identidad completa se conserva el literal `cli` y se emite **un** aviso por un `io.Writer` propio, solo cuando el registro se ha escrito de verdad. La salida estándar y el JSON no cambian (requisitos de reserva con aviso y de forma del JSON de la spec `change-actor-identity`).

`axiom.yaml` no declara `technical_context` (solo `workspace`, `git`, `workflow`, `roles` y `governance`), así que este cambio no produce `context-impact.md`.

## Decisiones de arquitectura

### D1. Paquete hoja `internal/gitidentity` sobre `gitcommon`

**Elección**: paquete nuevo `internal/gitidentity` con un único fichero de producción, `gitidentity.go`, que importa solo la biblioteca estándar e `internal/gitcommon`. Lee cada clave con `gitcommon.RunWithTimeout(ctx, dir, resolveTimeout, "config", "--get", key)`.

- `git config --get` lee todas las fuentes de configuración aplicables desde `dir` (sistema, global, local del repositorio e `includeIf` según el directorio) y devuelve el último valor si la clave tiene varios. Código 1 significa «no existe»; cualquier otro código distinto de cero, un error de ejecución (git ausente del `PATH`, directorio inexistente o relativo, tiempo agotado) o una salida vacía se tratan como clave ausente. No se leen `GIT_AUTHOR_*`/`GIT_COMMITTER_*` ni se usa `git var`, que puede inventar `usuario@host`.
- `gitcommon.RunWithTimeout` (`internal/gitcommon/exec.go`) ya aporta directorio absoluto y existente, retirada de las variables que redirigen el repositorio (`redirectingVariables`: `GIT_DIR`, `GIT_WORK_TREE`...), `LC_ALL=C`, límite de salida y observador de pruebas. `resolveTimeout` es una constante de 5 s: la identidad es opcional y no debe retener un `gate record` 30 s (el tiempo por defecto de `Run`).
- Dos invocaciones (una por clave) en lugar de una `--get-regexp`: la salida de `--get` es el valor en bruto y no hay que analizar líneas `clave valor` ni valores con saltos de línea.

**Alternativas consideradas**: (B) `gitcommon.ConfigValue` y el formateo en `cli`; (C) reutilizar `reviewAuditActor` (`internal/cli/review_audit_actor.go:26-41`); (D) resolver dentro de `internal/kickoff`.
**Justificación**: (B) deja la política de identidad dentro de `cli`, donde C-07-b y C-07-e (otros paquetes) no la pueden reutilizar sin duplicarla, y meter el formateo en `gitcommon` mezcla política con ejecución. (C) tiene otra semántica (reserva por email suelto, `GIT_AUTHOR_EMAIL` y el literal `gentle-ai-self-recovery@localhost`, ligada a vínculos de recuperación de RDD) y ejecuta `exec.CommandContext` directo; tocarla arrastra contratos de review a un cambio S. (D) rompe la pureza de `kickoff`, que hoy no ejecuta git, y de `ToKickoff`, que usan pruebas y e2e (`e2e/organicruntime/sdd_consent_wiring_e2e_test.go:77`). El grafo queda `cli → gitidentity → gitcommon`, sin ciclos: `gitcommon` no importa ningún paquete interno.

### D2. API: identidad saneada, sin error

**Elección**: la resolución devuelve un valor, nunca un error; el saneado y el formato son funciones puras.

```go
package gitidentity

const (
	KeyName  = "user.name"
	KeyEmail = "user.email"
	// MaxPartRunes es el tope de cada parte tras el saneado: el actor
	// "Nombre <email>" nunca supera 2*120+3 = 243 runes (< 256).
	MaxPartRunes = 120
)

// Identity es la identidad git saneada. Una parte vacía está ausente.
type Identity struct {
	Name  string
	Email string
}

// New sanea name y email con Sanitize. Pura.
func New(name, email string) Identity

// Complete dice si las dos partes están presentes.
func (i Identity) Complete() bool

// Actor devuelve "Nombre <email>" y true si la identidad está completa;
// "" y false si no.
func (i Identity) Actor() (string, bool)

// Missing devuelve las claves ausentes en orden fijo (KeyName, KeyEmail).
func (i Identity) Missing() []string

// Sanitize sanea una parte; "" significa ausente. Pura.
func Sanitize(value string) string

// Resolve lee user.name y user.email de la configuración de git desde dir
// (absoluto y existente) y devuelve New(name, email). Nunca falla.
func Resolve(ctx context.Context, dir string) Identity
```

**Alternativas consideradas**: `Resolve(ctx, dir) (Identity, bool)` o `(Identity, error)`; una identidad parcial que registra lo que haya.
**Justificación**: un `error` invitaría a cada llamador a decidir qué hacer con él, y la decisión ya está fijada (nunca bloquear); `Complete()` y `Missing()` dan al CLI todo lo que necesita para el aviso. La identidad parcial se trata como ausente (decisión de la propuesta): el plan solo define `Nombre <email>`.

### D3. Saneado de cada parte

**Elección**: `Sanitize` aplica, en este orden, el que fija la spec:

| Paso | Regla | Resultado si falla |
|---|---|---|
| 1 | `strings.TrimSpace` (retira también el salto final que imprime git) | — |
| 2 | Rechazo: UTF-8 no válido (`!utf8.ValidString`) o cualquier rune con `unicode.IsControl` (C0 con CR, LF y TAB, DEL, C1 y ESC) | `""` (ausente) |
| 3 | Retirada de `<` y `>` (`strings.NewReplacer("<", "", ">", "")`) | — |
| 4 | `strings.TrimSpace` | — |
| 5 | Truncado a `MaxPartRunes` runes y `strings.TrimRightFunc(..., unicode.IsSpace)` | — |

Una parte vacía tras el paso 5 es ausente. El formato es `Name + " <" + Email + ">"`. Como `<` y `>` se retiran de las dos partes, el actor contiene exactamente un `<` y un `>`, los que delimitan el email, y nunca coincide con los valores reservados `cli` ni `inferred` (contiene siempre `" <"`), así que la exención de `ValidateForSeal` (`internal/kickoff/schema.go:149-155`) no se puede activar por accidente. El valor de `--actor` explícito no se sanea.

**Alternativas consideradas**: eliminar los caracteres de control en lugar de rechazar la parte; tratar un valor demasiado largo como ausente; aceptar UTF-8 no válido.
**Justificación**: un CR/LF en `user.name` indica una configuración rota; quitarlo silenciosamente registraría un nombre que nadie configuró, mientras que rechazarlo lo hace visible con el aviso. El rechazo de los controles también impide secuencias de escape de terminal en `gate show`, que imprime el actor (`internal/cli/sdd_gate.go:240`). El truncado a 120 runes es lo que fija la spec. El UTF-8 no válido no lo regula la spec: yaml.v3 lo serializaría como `!!binary` (ilegible en el artefacto), por eso se trata igual que un carácter de control; se anota en Preguntas abiertas para que la spec lo recoja si se quiere.

### D4. Valor de reserva como constante del dominio

**Elección**: constante nueva en `internal/kickoff/types.go`, junto a `SealedByInferred` (`:80`):

```go
// FallbackActor es el actor y el sellador que se registran cuando no hay
// identidad git completa ni --actor explícito (change-actor-identity).
const FallbackActor = "cli"
```

`SealArgs.ToKickoff` (`internal/kickoff/args.go:375`) usa `SealedBy: FallbackActor` en lugar del literal y su comentario explica que el adaptador CLI la sustituye por la identidad git; sigue siendo pura. Los comentarios de `Kickoff.SealedBy` (`types.go:146`) y `GateRecord.Actor` (`types.go:191`) documentan el valor por defecto (identidad git `Nombre <email>`), la reserva `cli`, `inferred` en el retro-sellado y que el valor no se valida. Ningún tipo cambia.

**Alternativas consideradas**: dejar el literal `"cli"` repetido en `args.go` y en `cli`; mover la constante a `cli`.
**Justificación**: hoy el literal vive en dos sitios (`args.go:375` y `sdd_gate.go:102`) y este cambio añade un tercero (el aviso). Una constante en el dominio, al lado de la otra marca de sellado, evita la deriva y no introduce comportamiento: `ToKickoff` devuelve lo mismo byte a byte.

### D5. Costura y aviso en el adaptador CLI (`internal/cli/sdd_actor_identity.go`)

**Elección**: fichero nuevo en `internal/cli` con dos variables de paquete, siguiendo los patrones existentes `var sealKickoffFile = kickoff.Seal` (`sdd_kickoff.go:122-123`, costura de prueba) y `var sealBranchWarnings io.Writer = os.Stderr` (`sdd_kickoff_branch.go:32`):

```go
// resolveActorIdentity es la costura de prueba sobre gitidentity.Resolve.
var resolveActorIdentity = gitidentity.Resolve

// actorIdentityWarnings recibe el aviso de identidad git incompleta. Es
// propio: no comparte writer con los avisos de rama de seal ni de los
// escritores, cuyas pruebas afirman cero avisos.
var actorIdentityWarnings io.Writer = os.Stderr

// actorTarget nombra el campo que se rellena y si el verbo admite --actor.
type actorTarget struct {
	field       string // "actor" o "sealed_by"
	actorFlag   bool   // true en gate record
}

var (
	gateActorTarget = actorTarget{field: "actor", actorFlag: true}
	sealActorTarget = actorTarget{field: "sealed_by"}
)

// defaultActor resuelve la identidad git desde workspaceRoot. Devuelve el
// actor ("Nombre <email>" o kickoff.FallbackActor) y el aviso que hay que
// emitir si el registro llega a escribirse ("" con identidad completa).
func defaultActor(workspaceRoot string, target actorTarget) (actor, warning string)

// emitActorIdentityWarning escribe warning en actorIdentityWarnings si no
// está vacío (una línea, Fprintln).
func emitActorIdentityWarning(warning string)
```

`defaultActor` llama a `resolveActorIdentity(context.Background(), workspaceRoot)`, igual que `applySealBranchPolicy` usa `context.Background()` (`sdd_kickoff_branch.go:44`).

**Texto exacto del aviso** (una línea; `<raíz>` es `pathquote.Quote(workspaceRoot)`; `<faltan>` es `falta user.name`, `falta user.email` o `faltan user.name y user.email`; `<órdenes>` une con ` y ` las órdenes de las claves ausentes, `git config --global user.name "<nombre>"` y `git config --global user.email "<email>"`):

- `gate record`: `Aviso: la identidad git no está completa en <raíz> (<faltan>): se registra actor "cli". Configúrala con <órdenes>, o indica --actor.`
- `kickoff seal`: `Aviso: la identidad git no está completa en <raíz> (<faltan>): se registra sealed_by "cli". Configúrala con <órdenes>.`

Ejemplo: `Aviso: la identidad git no está completa en C:\repos\x (falta user.email): se registra actor "cli". Configúrala con git config --global user.email "<email>", o indica --actor.`

**Alternativas consideradas**: reutilizar `sealBranchWarnings`/`writerBranchWarnings`; un campo `warnings` en el JSON; recomendar `git -C <raíz> config` (configuración local).
**Justificación**: reutilizar los writers de rama rompería las pruebas que afirman cero avisos de rama (`sdd_kickoff_branch_test.go:220-233`, `sdd_kickoff_test.go:592-604`) en un entorno sin identidad. El JSON no cambia por decisión de la propuesta. `git -C <raíz> config` falla si la raíz no es un repositorio; `--global` funciona siempre y es la configuración habitual. La raíz se nombra para que en `multirepo` se sepa desde qué repositorio se leyó.

### D6. Inyección en `gate record`

**Elección**: en `runSDDGateRecord` (`internal/cli/sdd_gate.go:100-103`) se sustituye el literal:

```go
actor, actorWarning := parsed.Actor, ""
if actor == "" {
	actor, actorWarning = defaultActor(workspaceRoot, gateActorTarget)
}
```

y, justo después de que `kickoff.AppendGate`/`AppendGateChecked` (`:144-151`) terminen sin error, `emitActorIdentityWarning(actorWarning)`. La resolución ocurre después de la guarda de rama, de la carga del kickoff y de las comprobaciones de roster (`:66-98`): un rechazo previo no lanza git. Con `--actor` explícito no se llama a `resolveActorIdentity` ni se avisa.

La línea de ayuda de `record` (`:48`) conserva su forma y se añade una línea debajo: `    --actor: si se omite, la identidad git configurada ("user.name <user.email>") en la raíz del workspace; sin ella, "cli"`.

**Alternativas consideradas**: avisar en el momento de resolver.
**Justificación**: si la compuerta se rechaza después (evidencia no confirmada, `--hand-off` sin artefactos confirmados), no se ha registrado nada y un aviso «se registra actor "cli"» sería falso.

### D7. Inyección en `kickoff seal`

**Elección**: en `runSDDKickoffSeal`, dentro de la rama sin `--infer`, tras `applySealBranchPolicy` (`internal/cli/sdd_kickoff.go:90`) y antes de cerrar la rama:

```go
sealing.SealedBy, sealWarning = defaultActor(loc.WorkspaceRoot, sealActorTarget)
```

y, tras `sealKickoffFile` (`:113`), `if wrote { emitActorIdentityWarning(sealWarning) }`. La rama `--infer` no se toca: sigue escribiendo `inferred` desde `InferKickoff` y no lanza git. Se resuelve desde `loc.WorkspaceRoot` y no desde `changeRoot`, porque la carpeta del cambio puede no existir aún (`resolveSealChangeRoot`, REQ-29.12) y `gitcommon.Run` exige un directorio existente; `WorkspaceRoot` es además el repositorio donde se escribe el artefacto (en `multirepo`, el de specs: `Location.WorkspaceRoot` es el padre del planning home, `internal/workspace/locate.go:47-48`).

**Alternativas consideradas**: resolver antes de `applySealBranchPolicy`; resolver dentro de `ToKickoff`; avisar aunque el kickoff ya existiera.
**Justificación**: un rechazo por topología o por rama no debe lanzar git ni avisar. `ToKickoff` debe seguir pura (D4). Con un kickoff ya sellado, `kickoff.Seal` devuelve al ganador sin reescribirlo (`wrote == false`): no se registra nada, así que no hay nada que avisar.

### D8. Hermeticidad de las pruebas de `internal/cli`

**Elección**: el `TestMain` de `internal/cli` (`internal/cli/protocol_probe_test.go:52`) fija `resolveActorIdentity` a un resolvedor determinista que devuelve `gitidentity.New("Prueba Axiom", "prueba@ejemplo.invalid")`. Las pruebas nuevas lo sustituyen localmente con ayudantes de guardar y restaurar (`stubActorIdentity`, `forbidActorIdentity`, `useRealActorIdentity`) y capturan el aviso con `captureActorIdentityWarnings`, igual que `captureSealWarnings` (`sdd_kickoff_branch_test.go:34-41`).

**Alternativas consideradas**: dejar la resolución real en todas las pruebas existentes; aislar git con variables de entorno en `TestMain`.
**Justificación**: el `TestMain` ya convierte en deterministas otras dependencias accidentales del entorno (`LookPathOverride`, `verifyEngramVersion`; el comentario de la función lo justifica así). Sin el resolvedor fijo, cada una de las decenas de pruebas existentes de `seal` y `gate record` lanzaría dos procesos git más (lento en Windows) y su resultado dependería de la configuración git de la máquina. La resolución real se cubre de extremo a extremo en pruebas concretas con `useRealActorIdentity` y git aislado.

### D9. Prueba de arquitectura

**Elección**: `TestArchitectureOnlyExecGoLaunchesProcesses` (`internal/gitcommon/architecture_test.go:106-153`) añade `collectSources(t, filepath.Join("..", "gitidentity"), "internal/gitidentity")` con un comentario que cita la capacidad `change-actor-identity`, e `"internal/gitidentity/gitidentity.go"` a la lista de etiquetas que el recorrido debe ver.

**Alternativas consideradas**: no añadirlo (la convención basta).
**Justificación**: el paquete existe para ejecutar git; la garantía de que lo hace solo con `gitcommon.Run` (sin `os/exec`) es exactamente la que fija esa prueba para los demás lectores de git (`techcontext`, `docpaths`, `sddpublish`). La etiqueta en la lista evita que el recorrido sea vacuo si el fichero se renombra.

## Flujo de datos

```
gate record (sin --actor)                    kickoff seal (sin --infer)
  ParseGateRecordArgs                          ParseSealArgs
  resolveGovernanceChangeRoot -> loc           resolveSealChangeRoot -> loc
  guarda de rama, kickoff, roster              ApplyTopology, ToKickoff (SealedBy = FallbackActor)
        |                                      applySealBranchPolicy
        v                                            |
  defaultActor(loc.WorkspaceRoot, gate)  <-----------+-- defaultActor(loc.WorkspaceRoot, seal)
        |
        v
  resolveActorIdentity == gitidentity.Resolve
        |  gitcommon.RunWithTimeout(dir, "config", "--get", "user.name")
        |  gitcommon.RunWithTimeout(dir, "config", "--get", "user.email")
        v
  gitidentity.New -> Sanitize x2 -> Identity
        |
   Complete? --si--> actor = "Nombre <email>", aviso = ""
        |
        no --> actor = "cli", aviso = "Aviso: ..." (claves de Missing())
        v
  GateRecord.Actor = actor                     sealing.SealedBy = actor
  AppendGate(...)  ok?                          Seal(...) -> wrote?
        |                                            |
        v                                            v
  emitActorIdentityWarning(aviso)              if wrote: emitActorIdentityWarning(aviso)
  -> actorIdentityWarnings (stderr)            -> actorIdentityWarnings (stderr)
  stdout / JSON sin cambios                    stdout / JSON sin cambios
```

`gate record --actor X`: `actor = X`, sin `defaultActor`, sin git, sin aviso. `seal --infer`: `InferKickoff` (`sealed_by: inferred`), sin `defaultActor`.

## Cambios por fichero

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/gitidentity/gitidentity.go` | Crear | `Identity`, `New`, `Sanitize`, `Complete`, `Actor`, `Missing`, `Resolve` sobre `gitcommon.RunWithTimeout` (D1-D3) |
| `internal/gitidentity/gitidentity_test.go` | Crear | Tabla de saneado y formato (puras) y resolución con git aislado (ver Estrategia de pruebas) |
| `internal/kickoff/types.go` | Modificar | Constante `FallbackActor = "cli"`; comentarios de `Kickoff.SealedBy` (`:146`) y `GateRecord.Actor` (`:191`) |
| `internal/kickoff/args.go` | Modificar | `ToKickoff` usa `FallbackActor` (`:375`) y documenta que el adaptador lo sustituye; sin cambio de comportamiento |
| `internal/kickoff/schema_test.go` (o `ledger_test.go`) | Modificar | Lectura y validación de registros con `cli`, `maintainer`, `release-manager`, `inferred` y `Nombre <email>`, y round-trip YAML de un actor con acentos, `#` y `:` |
| `internal/cli/sdd_actor_identity.go` | Crear | Costura `resolveActorIdentity`, writer `actorIdentityWarnings`, `actorTarget`, `defaultActor`, `emitActorIdentityWarning` y texto del aviso (D5) |
| `internal/cli/sdd_gate.go` | Modificar | Actor por defecto (`:100-103`), aviso tras `AppendGate` (`:144-151`) y línea de ayuda de `--actor` (`:48`) (D6) |
| `internal/cli/sdd_kickoff.go` | Modificar | `SealedBy` en la rama sin `--infer` tras `applySealBranchPolicy` (`:90`) y aviso si `wrote` (`:113`) (D7) |
| `internal/cli/protocol_probe_test.go` | Modificar | `TestMain` fija `resolveActorIdentity` a una identidad determinista (D8) |
| `internal/cli/sdd_actor_identity_test.go` | Crear | Pruebas de gate, seal, reserva, aviso, JSON, multirepo, round-trip y extremo a extremo |
| `internal/gitcommon/architecture_test.go` | Modificar | Escanea `internal/gitidentity` y exige ver `gitidentity.go` (D9) |

No se tocan: `internal/kickoff/schema.go` (sin validación nueva), `internal/kickoff/infer.go`, `review_audit_actor.go`, guías de agente, goldens, specs vivas ni artefactos existentes.

## Interfaces / Contratos

- **API Go**: la de D2 (`internal/gitidentity`) y la de D5 (no exportada, `internal/cli`). Nueva constante exportada `kickoff.FallbackActor`.
- **Artefactos**: `gates.yaml` `actor` y `kickoff.yaml` `sealed_by` pasan a llevar `Nombre <email>` por defecto. yaml.v3 emite un escalar plano o entrecomillado según haga falta (`#` precedido de espacio y `: ` obligan a comillas) y lo relee sin pérdida. Un binario anterior lo lee igual: el campo es `string` sin validación de valor.
- **CLI**: sin banderas nuevas; `seal` sigue rechazando `--actor` y `--sealed-by` como `bandera desconocida` (`internal/kickoff/args.go:250-251`). Una línea nueva en la ayuda de `gate`. Stdout y JSON idénticos; el aviso solo por stderr.

## Estrategia de pruebas

TDD estricto desactivado (`openspec/config.yaml`: `strict_tdd: false`); se escriben las pruebas con el código y se observan en rojo las de comportamiento nuevo cuando sea práctico. Toda prueba que ejecute git real aísla el entorno: `HOME`/`USERPROFILE` temporales, `GIT_CONFIG_GLOBAL` a un fichero temporal (vacío o con identidad), `GIT_CONFIG_NOSYSTEM=1` y, para el directorio no repositorio, `GIT_CEILING_DIRECTORIES` al padre del temporal; se omite con `-short` y sin git, como `isolateIntegrateGit` (`internal/cli/gittest_helpers_test.go:16-34`).

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unidad (`gitidentity`, puras) | Recorte; CR, LF, TAB, ESC y UTF-8 no válido dejan la parte ausente; retirada de `<`/`>` (`Ana <admin> Pérez` → `Ana admin Pérez`); truncado a 120 runes con nombre y email de 500; parte solo de espacios ausente; `Actor()` con un solo `<` y `>`; `Missing()` en orden fijo para ninguna, solo nombre, solo email | Tabla `TestSanitize`, `TestIdentityActorAndMissing` |
| Unidad (`gitidentity`, git real) | Identidad local; local prevalece sobre global; directorio no repositorio con identidad global; `GIT_AUTHOR_*` definidas y config vacía → ausente; `GIT_DIR` apuntando a otro repositorio con otra identidad no redirige; git fuera del `PATH` (`t.Setenv("PATH", t.TempDir())`) → ausente sin pánico; directorio relativo → ausente | `TestResolve*` con repositorios temporales (`git init`, `git config`) |
| Unidad (`kickoff`) | Registros anteriores (`cli`, `maintainer`, `release-manager`, `inferred`, `Nombre <email>`) cargan y validan; un registro nuevo convive con uno `cli`; round-trip de `Íñigo: Gutiérrez # Zufia <inigo@example.com>` por `AppendGate`/`LoadGates` y `Seal`/`Load`; `ToKickoff` sigue devolviendo `cli` | Extensión de `schema_test.go`/`ledger_test.go` |
| Integración (`cli`) | Gate con identidad completa (actor, sin aviso); `--actor` explícito con `forbidActorIdentity` (falla si se llama); reserva `cli` con aviso para ninguna, solo nombre y solo email (una línea, `Aviso: `, claves nombradas, `--actor` sugerido); JSON de gate con y sin identidad con el mismo conjunto de claves y sin `actor`/`warnings`; seal con identidad y con reserva (sin `--actor` en el aviso); `seal --infer` con `forbidActorIdentity` sigue escribiendo `inferred`; re-seal sobre kickoff existente no avisa; `seal --actor x` y `--sealed-by x` rechazados sin escribir; multirepo (`workspacetest.Canonical`): el resolvedor recibe la raíz del repositorio de specs; gate y seal con la misma identidad son idénticos; avisos de rama siguen vacíos; ayuda de `gate` menciona la identidad git | `sdd_actor_identity_test.go` con `stubActorIdentity`/`captureActorIdentityWarnings` |
| Extremo a extremo (`cli`, git real) | `useRealActorIdentity`: seal en `embeddedSealProject` (`sdd_kickoff_branch_test.go:45`) con `user.name`/`user.email` locales; gate en un workspace no repositorio con identidad en el `GIT_CONFIG_GLOBAL` aislado | Dos pruebas, omitidas con `-short` |
| Arquitectura | `internal/gitidentity` no importa `os/exec` ni red | `TestArchitectureOnlyExecGoLaunchesProcesses` ampliada |

Comandos acotados (Windows es lento; no se compilan binarios sueltos, solo `go test`):

```
go test ./internal/gitidentity/...
go test ./internal/kickoff/... -run "Gate|Seal|Kickoff|Ledger|Schema|ToKickoff"
go test ./internal/cli/ -run "ActorIdentity|SDDGate|KickoffSeal|Seal"
go test ./internal/gitcommon/ -run Architecture
go vet ./internal/gitidentity/... ./internal/kickoff/... ./internal/cli/...
```

Antes de cerrar `apply`: `go test ./internal/gitidentity/... ./internal/kickoff/... ./internal/cli/... ./internal/gitcommon/...` y, en `verify`, `go test ./...` y `go vet ./...` (reglas de `verify` de `openspec/config.yaml`).

## Matriz de amenazas

El cambio añade un subproceso (`git config --get`) y escribe su salida en artefactos versionados. No hace commits, push ni PR, ni clasifica ficheros.

| Frontera | Casos adversos mínimos | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas tipo documentación | `requirements.txt`, `README.sh`, Markdown ejecutable | N/A: no se clasifica ni ejecuta ningún fichero | — | — |
| Selección de repositorio git | `-C`, ruta relativa, ruta absoluta, `GIT_DIR`/`GIT_WORK_TREE` heredadas | Aplicable | git se ejecuta con `cmd.Dir = loc.WorkspaceRoot` (absoluto, existente, resuelto por `workspace.Locate`) mediante `gitcommon.RunWithTimeout`, que rechaza rutas relativas y retira las variables que redirigen el repositorio; nunca `-C` ni el directorio del proceso. Ruta relativa o inexistente → identidad ausente, sin error | `TestResolveRelativeDirIsAbsent`, `TestResolveIgnoresRedirectingGitDir`, prueba multirepo de `cli` (el resolvedor recibe la raíz de specs) |
| Estado de commit | staged, `commit -a`, índice vacío | N/A: no se hace ningún commit (el commit de `--hand-off` existe ya y no cambia) | — | — |
| Estado de push | rama de seguimiento, primer push, refspec | N/A: sin push | — | — |
| Comandos de PR | `--head`, prefijo de entorno, comandos compuestos | N/A: sin automatización de PR | — | — |
| Salida del subproceso hacia artefactos y terminal (específica) | `user.name` con CR/LF, ESC, TAB, UTF-8 no válido, `<`/`>`, `#`/`:`, 500 caracteres; git ausente o colgado | Aplicable | Saneado de D3 (rechazo de controles y UTF-8 no válido, retirada de `<`/`>`, tope de 120 runes), serialización con yaml.v3, tiempo máximo de 5 s y límite de salida de `gitcommon`; cualquier fallo → `cli` con aviso | `TestSanitize` (casos de la columna), round-trip en `kickoff`, `TestResolveWithoutGitIsAbsent` |

## Migración / despliegue

- Sin migración de datos: no se reescribe ningún `gates.yaml` ni `kickoff.yaml`, activo o archivado. Los registros existentes (`cli`, `maintainer`, `release-manager`, `inferred`) se leen y validan igual, porque nada compara ni valida el valor del actor.
- Compatibilidad hacia atrás: un binario anterior lee `Nombre <email>` como cualquier otro `string`.
- Paquetes afectados: `internal/gitidentity` (nuevo), `internal/cli`, `internal/kickoff` (constante y comentarios), `internal/gitcommon` (solo prueba).
- Reversión: revertir la PR única. El actor y el sellador vuelven al literal `cli`, desaparecen `internal/gitidentity`, la costura y el aviso. Los artefactos ya escritos con `Nombre <email>` siguen siendo válidos para el binario revertido. No hace falta `axiom sync`: no cambia ningún recurso gestionado.
- Privacidad: el email entra en artefactos versionados; ya es público en los commits del mismo repositorio.

## Preguntas abiertas

Ninguna bloqueante.

- [x] La spec (escrita en paralelo) no regulaba el UTF-8 no válido; el diseño lo trata como carácter de control (parte ausente, D3). Resuelto: REQ-42.7 incluye ahora el escenario «UTF-8 no válido».
- [ ] El tamaño previsto (unas 550-700 líneas con pruebas) supera la estimación de la propuesta (150-250) y el presupuesto de 400 líneas; con `single-pr` la fase de tareas debe dejarlo marcado como excepción de tamaño.
- [ ] El prefijo REQ-42 puede colisionar con otras hijas de la ola: confirmar en `main` antes de archivar (riesgo heredado de la propuesta).
