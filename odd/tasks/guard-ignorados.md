# ODD: el aviso de `axiom branch guard` ignora los ficheros que git ignora (guard-ignorados)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/guard-ignorados.md`. Espejo Engram: `odd/guard-ignorados/tasks`.
> Rama: `fix/guard-ignorados` · worktree `C:\repos\axiom-wt\guard-ignorados\axiom`.

Tipo: bug

## Objetivo

Que el aviso de la guarda D (evento `post`) no salte al escribir un fichero que git ignora en su checkout. Los demás casos se mantienen y la guarda sigue fallando en abierto.

## Problema

El 2026-11-25, `axiom branch guard --agent claude-code` (hook `PostToolUse`) avisó dos veces al editar `C:\repos\axiom\.claude\settings.local.json` en `main`. Ese fichero está ignorado (`.gitignore:37`) y es configuración personal del checkout: escribirlo en `main` es lo correcto. `warnFile` (`internal/branch/engine/guard_warn.go`) avisa de cualquier fichero que no sea un documento de Axiom sin mirar si git lo ignora.

## Por qué

Un aviso que salta en un caso correcto es ruido y acostumbra a ignorar la guarda. El arreglo vale para cualquier fichero ignorado que se edite en `main` (configuración local, artefactos de compilación, cachés).

## Decisiones

- **Opción B: `git check-ignore` solo antes de avisar** (orquestador de la sesión de origen, 2026-11-25; la persona usuaria revisará la PR). La spec viva (RF-BRA-22, escenario «El hook no lanza git ni ningún otro proceso») prohibía lanzar procesos en cualquier escritura. Se descartó la opción A, un comparador de gitignore en Go que solo leyera ficheros: suponía más de 250 líneas, duplicaba la semántica de git (exclusiones por directorio, `info/exclude`, `core.excludesFile`, `**`, negaciones, padres excluidos, `core.ignoreCase`, índice) y corría el riesgo de divergir. Con B, el camino sin aviso (casi todas las escrituras) sigue sin procesos.
- **Consulta**: `git -C <raíz del checkout> check-ignore -q -- <ruta relativa>`, mediante `gitcommon.RunWithTimeout` (el único punto de ejecución de git permitido en `internal/branch/**`). Es una sola invocación, local y sin red, y se lanza únicamente cuando `warnFile` ya ha decidido avisar. No usa `--no-index`, así que un fichero versionado que coincida con un patrón no cuenta como ignorado y se sigue avisando.
- **Límite de tiempo**: la consulta usa como máximo la mitad del presupuesto que le queda a la guarda (1 s en total). Si git tarda, la consulta se corta y el aviso aún cabe en el presupuesto.
- **Fallo en abierto, hacia el comportamiento de antes**: si git no está, falla (código distinto de 0 y 1) o agota su límite, el fichero se trata como no ignorado y el aviso se emite como hasta ahora. La guarda nunca bloquea por esta consulta.
- **Redacción del escenario**: el orquestador propuso «el camino sin aviso no lanza ningún proceso». Se ha precisado como «sin motivo de aviso no se lanza ningún proceso», porque un fichero ignorado sí lanza la consulta (es lo que permite saber que no hay que avisar) y luego no avisa. Los caminos que se resuelven solo con ficheros siguen sin procesos: bloqueo, documentos, sin cambio activo, rama correcta y fuera de proyecto.
- **Documentos de Axiom ignorados**: el bloqueo (`pre`) de `openspec/changes/<cambio>/` y `odd/tasks/<cambio>.md` no consulta git y no cambia aunque un `.gitignore` los excluya. Es la garantía fuerte de la rama, su contrato es léxico más `HEAD`, y un `.gitignore` no debe poder desactivarla. En el evento `post` esos documentos ya no se avisan (su responsabilidad es la del bloqueo), así que nunca llegan a la consulta.

## Alcance autorizado

- `internal/branch/engine/guard_warn.go` y sus pruebas (`guard_warn_test.go`, comentarios de `guard_test.go` y `guard_hook_test.go`).
- Comentario de paquete de `internal/branch/engine/guard.go` y ayuda de `internal/cli/branch_guard.go`.
- Delta de RF-BRA-22 en `## Spec viva` e integración en `openspec/specs/change-branch-lifecycle/spec.md` con `axiom archive integrate --odd guard-ignorados`.

Fuera de alcance: el bloqueo de documentos (`pre`), los plugins de OpenCode y la instalación del hook (RF-BRA-23).

## Modo TDD

Desactivado: `axiom.yaml` no declara TDD y nadie lo ha pedido. Se ejecutan las comprobaciones funcionales habituales con `go test` del paquete, sin binarios sueltos (Cortex XDR) y sin `go test ./...`.

## Tareas

- [x] **T1 · Consulta `check-ignore` antes de avisar, con pruebas y spec viva** (ruta: en línea; unidad de unas 200 líneas sin motivo de delegación: no hay paralelismo ni presión de contexto)
  - `warnFile` recibe el contexto y, justo antes de devolver el aviso, descarta los ficheros ignorados por git (`ignoredByGit`).
  - Pruebas en `internal/branch/engine`, con un repositorio temporal en `main` y un cambio `in_place` registrado:
    - un fichero ignorado (`.claude/settings.local.json` excluido por `.gitignore`) no avisa;
    - un fichero versionado avisa, aunque coincida con un patrón (`git add -f`);
    - si git falla (índice corrupto) o no está en el `PATH`, se avisa;
    - un documento de Axiom ignorado por git se sigue bloqueando en `pre` sin lanzar procesos;
    - `TestGuardWarnLaunchesNoProcess`: cero procesos en los caminos silenciosos y exactamente una invocación `check-ignore` en el aviso.
  - Comentarios de paquete y ayuda de la CLI al día.
  - Delta MODIFIED de RF-BRA-22 en `## Spec viva` e integración con `axiom archive integrate --odd guard-ignorados`.
  - Comprobación: `go test -count=1 ./internal/branch/engine/` y `go test -count=1 -run BranchGuard ./internal/cli/`; `gofmt -l` y `go vet` del paquete.

## Criterio de terminado

- Pruebas en verde con `go test` de `internal/branch/engine`.
- Un commit en `fix/guard-ignorados` con el código, las pruebas, este documento y la integración de la spec.

## Entrega

Una sola PR (`single-pr`), con una previsión de unas 200-300 líneas. Push, PR y fusión los decide la persona usuaria.

## Spec viva

### Capacidad: change-branch-lifecycle

```openspec
## MODIFIED Requirements

### Requirement: `guard`: Hook Opcional que Bloquea los Documentos de Axiom y Avisa en el Resto (RF-BRA-22)

`axiom branch guard` DEBE ser el hook de la capa D que usan los runtimes con hooks. Recibe del runtime la herramienta y la ruta que va a escribir el agente, y su efecto depende de la ruta:

1. **Documentos de Axiom** (la carpeta de un cambio u `odd/tasks/`): DEBE **bloquear** (con el mecanismo de bloqueo del runtime) cuando el veredicto de RF-BRA-19 sea `wrong_branch`, con un mensaje que incluya la continuación de `ensure`. Rige en cualquier proyecto Axiom, con independencia de que haya un cambio registrado (supuesto S-18), y aunque git ignore el documento.
2. **Cualquier otro fichero** de un repositorio con un cambio activo registrado por `ensure` en el registro local (RF-BRA-15) y no cerrado por `done`: DEBE **avisar sin bloquear** cuando la rama del checkout que contiene el fichero no es la que corresponde al cambio. El aviso DEBE incluir la ruta del worktree correcto o, en `in_place`, la rama esperada; con varios cambios activos registrados en el repositorio, nombra los de todos. El repositorio de specs de un `multirepo` queda exento de este aviso. Un fichero que git ignora en su checkout tampoco se avisa: es configuración o resultado local que no viaja con la rama. Un fichero versionado no cuenta como ignorado aunque coincida con un patrón de exclusión.
3. En cualquier otro caso DEBE no hacer nada: sin cambio activo registrado en el repositorio, o fuera de un proyecto Axiom, termina sin bloquear, sin avisar y sin escribir.

DEBE leer la rama actual leyendo `HEAD` del repositorio, sin lanzar git. Mientras las lecturas de ficheros no pidan un aviso (bloqueo, documento de Axiom, sin cambio activo registrado, checkout en la rama de un cambio activo, fuera de un proyecto Axiom), NO DEBE lanzar ningún proceso. Solo cuando las lecturas piden un aviso, y antes de emitirlo, el hook PUEDE lanzar una única consulta local `git check-ignore`, sin red y dentro de su presupuesto de tiempo, para descartar un fichero ignorado; si esa consulta falla, git no está disponible o se agota su límite, el aviso se emite como si el fichero no estuviera ignorado. DEBE ser fail-open: ante cualquier error interno, configuración o registro ilegibles o expiración del presupuesto, NO DEBE bloquear ni avisar. Solo afecta a las escrituras del agente a través del runtime: NUNCA al editor humano ni a las órdenes de la CLI. Es la única capa, junto con la doctrina, que cubre el documento ODD. NO DEBE escribir ningún fichero. El canal no bloqueante por el que cada runtime entrega el aviso se decide en `sdd-design`.
(Previously: no lanzaba ningún proceso en ningún camino y avisaba también de los ficheros ignorados por git)
(Previously: bloqueaba solo las escrituras en la carpeta de un cambio u `odd/tasks/`, no avisaba en otros ficheros y consultaba git para las rutas protegidas)

#### Scenario: Bloquea un documento de Axiom en rama equivocada
- **DADO** un cambio con instantánea y el checkout en la rama de integración
- **CUANDO** el runtime invoca el hook para escribir `openspec/changes/x/proposal.md`
- **ENTONCES** el hook bloquea la escritura
- **Y** el mensaje incluye la continuación `axiom branch ensure --cwd <ruta> --change x`

#### Scenario: Permite un documento de Axiom en la rama correcta
- **DADO** el mismo cambio con el checkout en `feat/x`
- **CUANDO** el runtime invoca el hook para esa escritura
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: Avisa sin bloquear en un fichero de código en la rama equivocada
- **DADO** un monorepo con el cambio `x` activo y registrado por `ensure` (worktree en `<raíz de worktrees>/x/<repo>`) y el checkout principal en `main`
- **CUANDO** el runtime invoca el hook para escribir `internal/foo/foo.go` en el checkout principal
- **ENTONCES** el hook no bloquea la escritura
- **Y** emite un aviso que incluye la ruta `<raíz de worktrees>/x/<repo>` como worktree correcto

#### Scenario: Con varios cambios activos el aviso nombra los worktrees de todos
- **DADO** dos cambios activos `x` e `y` registrados en el mismo repositorio y el checkout principal en `main`
- **CUANDO** el runtime invoca el hook para escribir un fichero de código en el checkout principal
- **ENTONCES** el aviso incluye la ruta del worktree de `x` y la de `y`

#### Scenario: En modo `in_place` el aviso nombra la rama esperada
- **DADO** un cambio `x` registrado en modo `in_place` y el repositorio desplegado en `main`
- **CUANDO** el runtime invoca el hook para escribir un fichero de código
- **ENTONCES** el hook no bloquea y el aviso nombra la rama esperada `feat/x`

#### Scenario: Sin aviso si el fichero está en el worktree correcto
- **DADO** el cambio `x` registrado y un fichero de código dentro de su worktree, cuyo `HEAD` es `feat/x`
- **CUANDO** el runtime invoca el hook para escribirlo
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: Sin aviso en un fichero ignorado por git
- **DADO** el cambio `x` activo y registrado, el checkout principal en `main` y un `.gitignore` que excluye `.claude/settings.local.json`
- **CUANDO** el runtime invoca el hook para escribir `.claude/settings.local.json` en el checkout principal
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: Un fichero versionado avisa aunque coincida con un patrón de exclusión
- **DADO** el cambio `x` activo y registrado, el checkout principal en `main` y un fichero versionado que coincide con un patrón de `.gitignore`
- **CUANDO** el runtime invoca el hook para escribirlo en el checkout principal
- **ENTONCES** el hook no bloquea y emite el aviso

#### Scenario: Si la consulta a git falla, el aviso se emite
- **DADO** el cambio `x` activo y registrado, el checkout principal en `main` y una consulta `git check-ignore` que falla o un git que no está disponible
- **CUANDO** el runtime invoca el hook para escribir un fichero de código en el checkout principal
- **ENTONCES** el hook no bloquea y emite el aviso como si el fichero no estuviera ignorado

#### Scenario: Un documento de Axiom ignorado por git se sigue bloqueando
- **DADO** un proyecto Axiom con el checkout en la rama de integración y un `.gitignore` que excluye la carpeta del cambio `x`
- **CUANDO** el runtime invoca el hook para escribir `openspec/changes/x/proposal.md`
- **ENTONCES** el hook bloquea la escritura sin lanzar ningún proceso

#### Scenario: Sin cambio activo registrado en el repositorio no hace nada con el código
- **DADO** un proyecto Axiom sin ningún cambio activo registrado por `ensure` y el checkout en `main`
- **CUANDO** el runtime invoca el hook para escribir `internal/foo/foo.go`
- **ENTONCES** el hook termina sin bloquear ni avisar

#### Scenario: El repositorio de specs de un multirepo queda exento del aviso
- **DADO** un multirepo con un cambio activo registrado y el repositorio de specs en su rama de integración
- **CUANDO** el runtime invoca el hook para escribir un fichero que no es un documento de Axiom en el repositorio de specs
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: Inerte fuera de un proyecto Axiom
- **DADO** un directorio sin `axiom.yaml` ni `.axiom-workspace` que resuelva a uno, en él o en sus ancestros (aunque tenga un `openspec/`)
- **CUANDO** el runtime invoca el hook para cualquier escritura
- **ENTONCES** termina sin bloquear ni avisar
- **Y** no escribe ningún fichero

#### Scenario: Fail-open ante un error interno
- **DADO** un proyecto Axiom con `axiom.yaml` o registro local ilegibles
- **CUANDO** el runtime invoca el hook para escribir en una carpeta de cambio o en un fichero de código
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: Sin motivo de aviso no se lanza ningún proceso
- **DADO** un proyecto Axiom y un observador de procesos
- **CUANDO** el runtime invoca el hook para una escritura que se resuelve sin aviso con las lecturas de ficheros: un bloqueo o un documento de Axiom, sin cambio activo registrado, con el checkout en la rama de un cambio activo o fuera de un proyecto Axiom
- **ENTONCES** el observador no registra ninguna invocación de git ni de ningún otro proceso
- **Y** la rama actual se obtiene leyendo `HEAD`

#### Scenario: Antes de avisar, como mucho una consulta `git check-ignore`
- **DADO** el cambio `x` activo y registrado, el checkout principal en `main` y un observador de procesos
- **CUANDO** el runtime invoca el hook para escribir en el checkout principal un fichero de código o un fichero ignorado por git
- **ENTONCES** el hook avisa del fichero de código y no avisa del ignorado
- **Y** en cada escritura el observador registra como mucho una invocación de git, `check-ignore`, local y sin red

#### Scenario: El documento ODD está cubierto
- **DADO** un monorepo con el checkout en la rama de integración y la política por defecto
- **CUANDO** el runtime invoca el hook para escribir `odd/tasks/mi-feature.md`
- **ENTONCES** el hook bloquea con la continuación `axiom branch ensure --lane odd --change mi-feature`

#### Scenario: Solo afecta al agente a través del runtime
- **DADO** un proyecto con el hook instalado
- **CUANDO** una persona guarda un fichero desde su editor o ejecuta una orden de la CLI
- **ENTONCES** el hook no se ejecuta, porque solo lo invoca el runtime en las escrituras del agente y no se instala ningún hook de git ni observador de ficheros
```

## Progreso

- 2026-11-25 · Documento creado. Decisión B tomada por el orquestador.
- T1 · se cierra en el commit único de la feature, `fix(branch): no avisar en el guard de los ficheros que git ignora`. Contiene el código, las pruebas, este documento y la integración de la spec; su sha se comunica al cerrar porque un commit no puede citarse a sí mismo.
  - `warnFile(ctx, path)` llama a `ignoredByGit` justo antes de devolver el aviso: `gitcommon.RunWithTimeout(ctx, root, …, "check-ignore", "-q", "--", <relativa>)`, con un límite de la mitad del plazo que queda (500 ms si no hay plazo). Cualquier error se trata como no ignorado.
  - Pruebas nuevas: `TestGuardWarnSkipsFilesIgnoredByGit` (por nombre, por directorio, versionado con `add -f`, no ignorado, parche mixto), `TestGuardWarnWarnsWhenTheIgnoreCheckFails` (índice corrupto), `TestGuardWarnWarnsWithoutGit` (`PATH` vacío) y `TestGuardBlocksAnAxiomDocumentEvenIfGitIgnoresIt`. `TestGuardWarnLaunchesNoProcess` pasa a llamarse `TestGuardWarnLaunchesOnlyOneCheckIgnoreBeforeWarning`: cero procesos sin motivo de aviso y exactamente una invocación `check-ignore -q -- <relativa>` en la raíz del checkout, tanto si avisa como si el fichero está ignorado.
  - `axiom archive integrate --odd guard-ignorados`: `change-branch-lifecycle: composed` (spec e `INDEX.md`); repetirlo da `already_integrated`.
- RDD: no se usa.

## Verificación

- `gofmt -l internal/branch/engine internal/cli/branch_guard.go`: sin salida.
- `go vet ./internal/branch/engine/`: sin avisos.
- `go test -count=1 -run TestGuard ./internal/branch/engine/`: ok.
- `go test -count=1 -v` de las seis pruebas nuevas o modificadas: las seis en PASS, ninguna omitida.
- `go test -count=1 ./internal/branch/engine/` (paquete completo): ok (459 s).
- `go test -count=1 -run 'BranchGuard|GuardPopulation' ./internal/cli/`: la primera ejecución falló en `TestBranchGuardHelpDescribesTheHookAndExitsZero`, que exige el corte de línea `falla en\nabierto` en la ayuda. Se reordenó el párrafo sin cambiar el sentido y la nueva ejecución dio ok. `go vet ./internal/cli/`: sin avisos.
- No se ejecutaron pruebas de `internal/app`, `internal/dashboard`, `internal/update`, `internal/opencode`, `internal/components/gga`, `bench` ni `e2e`, ni `go test ./...` (restricción de Cortex XDR). Los plugins de OpenCode no cambian.

## Seguimientos

- `axiom archive integrate --odd` volvió a quitar la regla `---` que separa el requisito MODIFIED (RF-BRA-22) del siguiente (RF-BRA-23) en `openspec/specs/change-branch-lifecycle/spec.md`. Es el mismo fallo que anotó `fix-miner-frontmatter` y no se corrige aquí: solo la integración puede escribir en la spec.
- La ayuda de `axiom branch guard` dice «Código 2 … si bloquea; 0 en otro caso», pero con `claude-code` y `codex` un aviso también sale con código 2 (`writeGuardDecision`). Es anterior a este cambio y queda fuera de alcance.
