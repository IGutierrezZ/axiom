# ODD · alta-proyecto-ui (P4 de la hoja de ruta de visión de producto)

## Objetivo

Dar de alta un proyecto desde el dashboard con la configuración elegida y validada en la UI antes de escribir nada en disco. La configuración cubre:

- la topología y, en multirepo, la raíz y el repositorio de specs;
- los roles y sus repositorios;
- los adaptadores que se instalan, siempre en modo workspace;
- la review (RDD) como preferencia del usuario para ese proyecto.

## Problema y porqué

Auditoría `docs/audits/2026-11-03-vision-producto.md`, objetivo O4, huecos G4.1–G4.7. Todos están verificados al mapear el 2026-11-10.

- **G4.1. El alta no se valida.**
  - `hub.Init` (`internal/hub/init.go`) escribe `axiom.yaml` sin pasarlo por `workspace.ParseConfig` ni por `workspace.Validate`.
  - La UI solo comprueba que la ruta no esté vacía.
  - Las claves de rol se escriben sin comillas: una clave duplicada o con `:` produce un YAML roto, y eso se descubre después de escribir.
  - Cualquier fallo sale como HTTP 500.
- **G4.2. No hay campos de multirepo.**
  - Se escribe siempre `root: "."` y `specs_repository: openspec`.
  - El puntero `.axiom-workspace` solo se monta a mano.
- **G4.3. Roles y repositorios sin validar.**
  - Los repos de rol no se comprueban: ni que existan, ni que sean relativos, ni que estén dentro del proyecto.
  - `tech` no se puede editar.
- **G4.4. `gate_policy` fijo en el dashboard.**
  - `internal/dashboard/service.go:187` fija `GatePolicy: "blocking"` e ignora el valor real del rol.
  - Además, el alta escribe `advisory` para los roles no bloqueantes. `multirole` solo admite `blocking`, `deferred` y `optional` (`internal/multirole/detector.go:109`), así que esos proyectos fallan al detectar roles.
- **G4.5. La review es un interruptor de máquina.** No está en la UI web.
- **G4.6. La UI no elige ni instala adaptadores.**
  - `/api/ecosystem/sync` solo sincroniza lo ya instalado.
  - Ese endpoint hace `os.Chdir`, que afecta a todo el proceso.
- **G4.7. El modo workspace es el ámbito por defecto**, pero queda huella fuera de él. No se toca aquí.

## Decisiones

- **D2 (decidido por el usuario el 2026-11-10 y el 2026-11-11):** el RDD es una preferencia de cada usuario por proyecto y nunca se versiona.
  - Se guarda en el registro del hub (`~/.axiom/workspaces.json`, `WorkspaceRecord`), junto a las preferencias de worktree.
  - Precedencia:
    1. El «apagar» por clon gana.
    2. Si no lo hay, manda la preferencia del proyecto.
    3. Si tampoco la hay, manda el interruptor global.
  - Un repositorio sigue sin poder forzar el encendido.
- **Las siguientes las tomé por defecto. Se revisan en la parada de plan:**
  - **`gate_policy`:**
    - Un rol no bloqueante se escribe como `deferred`, no como `advisory`.
    - El alta valida con el vocabulario de `multirole`.
    - Migrar los `advisory` ya escritos queda fuera de alcance (es un seguimiento).
  - **Adaptadores:**
    - Son opcionales. Si no se elige ninguno, el proyecto queda sin adaptadores y el doctor de P3 lo refleja.
    - Los agentes sin modo workspace (Pi, OpenClaw, Windsurf, Antigravity, Trae) se muestran deshabilitados con el motivo.
  - **Repositorio de specs en multirepo:** el alta crea la carpeta si no existe, pero no hace `git init`. El doctor de P3 avisa si no es git.
  - **CLI:** `axiom init` usa la misma validación previa. Las configuraciones que hoy escribe rotas pasan a rechazarse con la lista de errores.
  - **Instalación en proceso con raíz explícita:** nada de `os.Chdir`.

## Alcance autorizado

- `internal/hub/**`: alta, validación previa y preferencias.
- `internal/workspace/**`: solo lo que necesite la validación previa, como un FS superpuesto.
- `internal/dashboard/**`: servicio, tipos, handlers, assets y pruebas.
- `internal/cli/`, en tres partes:
  - el núcleo de instalación con raíz explícita;
  - `review_mode.go`;
  - `axiom init` en `cmd/axiom/main.go`, solo para la validación.
- `internal/reviewtransaction/rdd_mode.go`: el nuevo origen de la preferencia por proyecto.
- `docs/usage.md` y este documento.

Fuera de alcance:

- Editar roles después del alta (resto de G4.3).
- La huella fuera del workspace (G4.7).
- El `os.Chdir` de `RunSync` en `/api/ecosystem/sync`, que ya está anotado como seguimiento.
- Migrar los `advisory` existentes.

## Configuración

- **TDD:** off. `axiom.yaml` no lo configura, así que se hacen las comprobaciones funcionales normales.
- **RDD:** off (`axiom review mode status`: decidido por defecto).
- **Ejecutor de pruebas:** `go test` en proceso. **Prohibido compilar binarios sueltos** (Cortex).
  - El JS se prueba con Node mediante `runNodeModules`, solo para módulos de `js/core/`.
  - `internal/cli` entero tarda más de una hora aquí, así que se acota con `-run`.
- **Entrega:** `single-pr` con `size:exception` y un commit por tarea (preferencia del usuario). Previsión: entre 2.300 y 3.000 líneas con pruebas.

## Tareas

- [x] **T1 · `gate_policy` correcto (G4.4).**
  - `GetWorkspace` devuelve el `GatePolicy` real del rol, con `blocking` por defecto.
  - El alta escribe `deferred` en lugar de `advisory`.
  - Pruebas en el dashboard y en `hub`.
- [x] **T2 · Validación previa en `hub` (G4.1, G4.3).**
  - `(*hub.Initializer).ValidateInit(opts)` genera la configuración y la pasa por `workspace.ParseConfig` y por `workspace.Validate`, con un FS superpuesto que da por creadas las carpetas que el alta va a crear.
  - Repos relativos y dentro del proyecto. Claves de rol entre comillas.
  - `gate_policy` con el vocabulario de `multirole`.
  - Devuelve una lista de violaciones, no un único error.
  - `Init` la llama antes de cualquier escritura, también desde `axiom init`.
- [x] **T3 · Errores de validación en la API y la UI.**
  - `handleProjectsInit` devuelve 400 con la lista de violaciones.
  - El modal las muestra.
  - La validación en cliente (claves duplicadas, campos obligatorios vacíos) va a un módulo de `js/core/`, probado con Node.
  - Se añade el campo `tech` por rol (parte de G4.3).
- [x] **T4 · Campos de multirepo (G4.2).**
  - `root` y `specs_repository` en `InitOptions` y en `ProjectInitRequest`.
  - Se escribe el puntero `.axiom-workspace` y se verifica con `workspace.LocateAt`.
  - Los campos del modal solo se ven con topología multirepo.
- [x] **T5 · Elección e instalación de adaptadores (G4.6).**
  - Endpoint de catálogo: id, nombre y si admite modo workspace.
  - Núcleo de instalación en proceso con raíz explícita, sin `os.Chdir`. El estado del proyecto registra los agentes instalados.
  - Selector en el modal.
  - Pruebas con la raíz del proyecto nuevo y el cwd en otra carpeta.
- [x] **T6 · RDD por usuario y proyecto, mecanismo (G4.5, D2).**
  - Preferencia en `WorkspaceRecord`, con la precedencia de D2 en `ResolveRDDMode`.
  - `axiom review mode enable|disable|status --scope project`.
  - `status` informa del origen que decide.
  - Pruebas de precedencia, y de que `axiom.yaml` y el estado del proyecto no llevan nada.
- [x] **T7 · RDD en el alta y en la UI.**
  - Opción en el modal de alta.
  - El dashboard la muestra y la cambia para el proyecto abierto.
  - Protección de origen (`isLocalRequest`) en el POST.
- [ ] **T8 · Documentación y cierre.**
  - [x] `docs/usage.md`: alta desde la UI y RDD por proyecto.
  - [ ] PR. Al fusionar, marcar P4 y D2 en la auditoría.

## Criterios de aceptación

- Un alta con errores (clave duplicada, repo fuera del proyecto, `gate_policy` inválido, specs ausente en multirepo sin poder crearlo) no escribe nada y la UI muestra cada error.
- Un alta multirepo deja `axiom.yaml` en el repo de specs y el puntero en la carpeta maestra. `workspace.LocateAt` la resuelve.
- Los adaptadores elegidos quedan instalados en el proyecto nuevo en modo workspace, sin depender del cwd del proceso.
- Con el RDD encendido en un proyecto y apagado en el global, `axiom review mode status` en ese proyecto dice que está encendido y por qué. El «apagar» por clon sigue ganando. `axiom.yaml` no cambia.
- El dashboard muestra el `gate_policy` real de cada rol.
- `axiom init` sigue funcionando con configuraciones válidas.
- Pruebas acotadas en verde. Como mínimo: `go test ./internal/hub/... ./internal/workspace/... ./internal/dashboard/... ./internal/reviewtransaction/...` y `go test ./internal/cli/ -run '<filtro de la tarea>'`.

## Coordinación

T18–T20 de `odd/tasks/upstream-revision-nativa.md` tocan `rdd_mode.go` y `review_mode.go` y esperaban a D2. T6 cambia la resolución del modo en esos mismos ficheros. Quien retome T18–T20 debe partir de T6 fusionada y revisar la regla de T19 («si el global no es `on`, se ignora el fallo del override»).

## Progreso y evidencias

- 2026-11-10: mapa hecho (delegado, solo lectura).
- 2026-11-11: el usuario elige el registro del hub para el RDD por proyecto.
- 2026-11-11: T1 hecha (ruta: escritor delegado, commit 09ce1f7b). GetWorkspace devuelve el gate_policy real del rol (blocking por defecto); el alta escribe deferred en lugar de advisory; la etiqueta del modal ya no habla de advisory. Observado: go test ./internal/hub/... ./internal/workspace/... ./internal/dashboard/... -count=1: ok; go test ./internal/multirole/... : ok; go test ./cmd/axiom/ -run Init : ok; go vet y gofmt -l: limpios.
- 2026-11-11: T2 hecha (ruta: escritor delegado, commit 43b0e567). Nuevo hub.ValidateInit (internal/hub/init_validate.go): planifica el axiom.yaml, lo pasa por workspace.ParseConfig y workspace.Validate con un FS superpuesto (proyecto, openspec/ y repos de rol dados por existentes) y aplica reglas de claves repetidas (también por mayúsculas), repos absolutos o fuera del proyecto y vocabulario de gate_policy de multirole; devuelve *InitValidationError con todas las violaciones. Init lo llama antes de MkdirAll y de cualquier escritura (también axiom init); las claves de rol se escriben entre comillas cuando hace falta. Decisión: el alta no exige que existan los repos de rol (los traen las configuraciones adoptadas y los proyectos nuevos). Observado: go test ./internal/hub/... ./internal/workspace/... ./internal/dashboard/... -count=1: ok; go test ./internal/multirole/... : ok; go test ./cmd/axiom/ -run Init : ok; go vet y gofmt -l: limpios.
- 2026-11-11: T3 hecha (ruta: escritor delegado, commit 5ec3bfa5). handleProjectsInit devuelve 400 con {error, violations} ante *hub.InitValidationError y 500 solo para lo inesperado; el modal lista cada violación; la validación en cliente (ruta obligatoria, claves duplicadas, repos absolutos o fuera del proyecto) vive en js/core/projectform.js y se prueba con Node; nuevo campo tecnologías por rol (hub.RoleInput ya aceptaba tech, ahora se envía y se escribe). Observado: go test ./internal/hub/... ./internal/workspace/... ./internal/dashboard/... -count=1: ok; go test ./internal/multirole/... : ok; go test ./cmd/axiom/ -run Init : ok; go vet y gofmt -l: limpios. Sin verificar: aspecto visual del modal (no se abrió en navegador).
- 2026-11-11: T4 hecha (ruta: escritor delegado, commit f7610dc6). InitOptions.SpecsRepository y ProjectInitRequest.specs_repository: en multirepo el alta crea el repositorio de specs (mkdir, sin git init), escribe en él axiom.yaml y openspec/, deja el puntero .axiom-workspace (config: <repo>/axiom.yaml) en la carpeta maestra y verifica con workspace.LocateAt (un fallo se devuelve como error). inspectOwnMarkers recibe el planning home esperado: la protección contra un planning home ajeno sigue activa y repetir el alta es idempotente. ValidateInit cubre la ruta de specs (propia carpeta, absoluta, fuera, en monorepo) y rechaza añadirla a un axiom.yaml directo ya existente. El modal solo muestra el campo en multirepo, con validación en projectform.js. Decisiones: no hay campo root aparte (workspace.root no lo lee ningún resolutor; la carpeta maestra es la ruta del alta y el multirepo con repositorio propio no escribe root:); sin SpecsRepository (CLI incluido) el multirepo es el de siempre. Observado: go test ./internal/hub/... ./internal/workspace/... ./internal/dashboard/... -count=1: ok; go test ./cmd/axiom/ -run Init|Install : ok; go vet y gofmt -l: limpios.
- 2026-11-11: T5 hecha (ruta: escritor delegado, commit 31753528). Núcleo extraído de RunInstall: runInstall recibe la raíz ya resuelta (RunInstall resuelve la carpeta actual una sola vez y delega; newInstallRuntimeAt es la variante con raíz explícita) y cli.InstallInWorkspace(projectDir, args) lo invoca con --scope workspace sin os.Chdir ni leer el cwd; el estado del proyecto (InstalledAgents en .axiom/state.json) es el de axiom install --scope workspace. Elegido frente a RunSyncWithSelectionScoped porque es el mismo código de instalación y de persistencia del estado. Dashboard: GET /api/adapters (id, nombre, workspace, motivo), ProjectInitRequest.adapters validado antes de escribir (400 con violaciones junto a las del proyecto), instalación tras escribir con fallo parcial (200 con adapters.error, el proyecto no se deshace), casillas en el modal con los no-workspace deshabilitados. Observado: go test ./internal/hub/... ./internal/workspace/... ./internal/dashboard/... -count=1: ok; go test ./internal/cli/ -run Install|Scope|WorkspaceAgents -count=1: ok (335s); go test ./cmd/axiom/ -run Init|Install : ok; go vet y gofmt -l: limpios. Sin verificar: aspecto del modal; instalación real con el preset por defecto desde el dashboard (los tests la sustituyen o usan engram con preset custom).
- 2026-11-11: T6 hecha (ruta: escritor delegado, commit 5fd2d857). Preferencia de RDD por usuario y proyecto en el registro del hub (WorkspaceRecord.RDDMode, hub/rddpref.go: ProjectForRepo, LookupProjectRDD, SetProjectRDD; en multirepo los repos de rol y el de specs resuelven a la entrada de la carpeta maestra, gana el proyecto más profundo). reviewtransaction no importa hub (hub depende de reviewtransaction: habría ciclo): RDDGlobalMode gana Project, que rellena la capa CLI (readUserRDDMode); precedencia en ResolveRDDMode: off del clon, preferencia de proyecto (en ambos sentidos), global, apagado por defecto; RDDModeSourceProject y RDDModeStatus.Project. CLI: axiom review mode enable|disable|status --scope project e inherit --scope project (retira la preferencia; añadido porque la UI de T7 necesita heredar), error claro fuera de un proyecto registrado; los puntos de autorización, el stop hook y status usan la misma lectura; un hub ilegible falla cerrado nombrando el fichero. axiom.yaml y .axiom/state.json no se tocan (ProjectView sigue en blanco). Observado: go test ./internal/hub/... ./internal/workspace/... ./internal/dashboard/... ./internal/state/... -count=1: ok; go test ./internal/reviewtransaction/ -run RDD|Mode -count=1: ok; go test ./internal/cli/ -run ReviewMode|RDD|Install|WorkspaceAgents -count=1: ok (284s); go test ./cmd/axiom/ -run Review|Init : ok; go vet y gofmt -l: limpios. El paquete reviewtransaction entero supera el timeout de 10 min por TestAuthorityDispositionResumeCrashPositionMatrix (lento en este equipo, sin relación con el modo).
- 2026-11-11: T7 hecha (ruta: escritor delegado, commit c60fcebc). Alta: ProjectInitRequest.review_mode (inherit/on/off) se valida antes de escribir y se guarda en el hub tras escribir y registrar (fallo parcial en review_mode_error, sin deshacer el proyecto); el modal tiene el selector. Dashboard: GET/POST /api/project/review-mode sobre la raíz del proyecto abierto (sin cwd; 409 si no está registrado, 400 si el modo no es válido) y tarjeta Revisión (RDD) en Ecosistema con modo efectivo, fuente que decide y selector heredar/on/off; textos en js/core/reviewmode.js probados con Node. Protección de origen: POST /api/projects/init y POST /api/project/review-mode exigen isLocalRequest (403 con Origin o Host ajenos, sin escribir nada); los helpers de pruebas usan Host 127.0.0.1. Auditoría del cwd en la instalación workspace (sin cambios de código): las órdenes externas son npm install -g / brew / sudo npm -g de los adaptadores (agents/*/InstallCommand), go install del binario de engram y npm|pnpm add -g de CodeGraph más codegraph install --location global; engram setup no se lanza en ámbito workspace (run.go) y Pi/OpenClaw, los únicos con órdenes de instalación local (pi install, pi-engram init), no admiten workspace. Todas son globales o no leen el proyecto, así que no dependen del cwd; único residuo teórico: codegraph install --yes sin --location con un CodeGraph antiguo, que solo ocurre eligiendo ese componente, no con el --agent del alta. Observado: go test ./internal/hub/... ./internal/workspace/... ./internal/dashboard/... ./internal/state/... -count=1: ok; go test ./internal/reviewtransaction/ -run RDD|Mode : ok; go test ./internal/cli/ -run ReviewMode|RDD|InstallInWorkspace|WorkspaceAgents -count=1: ok; go test ./cmd/axiom/ -run Review|Init : ok; go vet y gofmt -l: limpios. Sin verificar: aspecto visual de la tarjeta y del selector.
- 2026-11-11: T8 (documentación) hecha (ruta: escritor delegado). docs/usage.md: nuevo apartado init (validación previa, deferred), alta de proyecto desde el dashboard dentro de ui (campos, validación previa con lista de errores, adaptadores en modo workspace, fallo parcial, protección de origen local) y RDD por usuario y proyecto junto al RDD de la instalación (--scope project, precedencia, hub, tarjeta del dashboard). Pendiente: el PR y marcar P4 y D2 en la auditoría al fusionar. También docs/trigger-rules.md: filas de review mode --scope project (enable, disable, inherit) y precedencia. Observado: lectura estructural de los ficheros; go test ./internal/app/ -run Documented|Invocation -count=1: ok (ejecuta las invocaciones documentadas; los comentarios en línea de un bloque de código rompían el analizador, así que los comentarios van en líneas aparte); no hay copia de trigger-rules que compare ninguna prueba.

## Seguimientos fuera de alcance

- **CodeGraph antiguo:** `codegraph install --yes` sin `--location` (rama de un CodeGraph anterior al contrato de destinos) podría depender del directorio actual. Solo ocurre al elegir ese componente; el alta no lo instala por sí sola.
- **Aspecto visual sin verificar:** el modal de alta (specs, adaptadores, RDD, lista de errores) y la tarjeta de Revisión (RDD) se probaron por lógica, API y estructura, no en un navegador.
- **Migrar los `gate_policy: advisory` ya escritos** por altas anteriores: el vocabulario de multirole no los admite y esos proyectos fallan al detectar roles.
- **Editar roles después del alta** (resto de G4.3).
- **`os.Chdir` de `RunSync`** en `/api/ecosystem/sync`, que afecta a todo el proceso.
- **T18–T20 de `odd/tasks/upstream-revision-nativa.md`** deben partir de T6 fusionada: cambia la resolución del modo en `rdd_mode.go` y `review_mode.go` y hay que revisar la regla de T19.

Plan stop: approved 2026-11-11; implementation continues
