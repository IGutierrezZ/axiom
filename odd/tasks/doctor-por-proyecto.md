# ODD · doctor-por-proyecto (P3 de la hoja de ruta de visión de producto)

## Objetivo

El doctor recibe la raíz del proyecto como parámetro y devuelve un resultado estructurado. El dashboard muestra sus comprobaciones reales al abrirse y cada vez que cambia el proyecto abierto. Cuando se cambia de proyecto, el resultado anterior deja de mostrarse.

Se añaden comprobaciones del proyecto:

- assets gestionados,
- adaptadores,
- topología,
- repositorios de rol,
- repositorio de specs.

## Problema y porqué

Auditoría `docs/audits/2026-11-03-vision-producto.md`, objetivo O3, huecos G3.1–G3.5. Todos están verificados al mapear.

- **G3.1. El dashboard no ve el resultado real del doctor.** `GetDoctorDiagnostics` (`internal/dashboard/service.go:1223-1294`) llama a `cli.RunDoctor` y nunca lee su salida.
  - `Healthy` solo pasa a `false` si `RunDoctor` devuelve error, y casi nunca lo hace.
  - Los avisos no cuentan para `Healthy`.
- **G3.2. El doctor no sabe qué proyecto mirar.** Resuelve el proyecto con `os.Getwd()` (`internal/cli/install_state_lock.go:46`).
  - Desde #223, el cwd del servidor es un directorio neutro (home o temporal). Por eso el doctor del dashboard nunca ve el proyecto abierto.
- **G3.3. Al cambiar de proyecto se sigue viendo el informe anterior.**
  - La caché de `overview.js` conserva los datos previos mientras refresca.
  - Una petición en vuelo del proyecto anterior puede sobrescribir la del nuevo.
  - El DTO no lleva el proyecto.
- **G3.4. Solo hay una comprobación de proyecto,** `workspace-axiom.yaml`, que mira `ws.Compliant`.
- **G3.5. El doctor y el dashboard hablan formatos distintos.**
  - `internal/doctor` tiene `Report`/`Result` con `pass/warn/fail`, sin etiquetas JSON.
  - El dashboard tiene un DTO propio con `ok/warning/error`.

## Decisiones

Las tomé por defecto y se presentan en la parada de plan.

- **Drift de assets en esta versión:** se compara el digest gestionado del binario (`assets.ManagedDigest()`) con `InstallState.ManagedAssetDigest` del estado del proyecto.
  - Si difieren, aparece un aviso con el remedio `axiom sync --scope workspace`.
  - No se construye un diff por fichero.
- **El dashboard muestra las comprobaciones globales y las del proyecto, en secciones separadas.**
  - Globales: herramientas, Engram, disco, estado global.
  - Del proyecto: el resto.
  - `Healthy` refleja los fallos de ambas.
- **Repos de rol:** se comprueba que existen y que son repositorios git.
  - Sin un cambio activo no se comprueba la rama.
  - Que falte un repositorio es un fallo; un repositorio que no es git es un aviso.
- **Sin red ni subprocesos nuevos** en las comprobaciones de proyecto. El sondeo existente de Engram se mantiene.
- **Ninguna comprobación depende del cwd del proceso:** nada de `os.Getwd` ni `os.Chdir` en el camino del doctor.

## Alcance autorizado

- `internal/doctor/**`.
- `internal/cli/doctor*.go` e `install_state_lock.go`, solo para parametrizar la raíz.
- `internal/dashboard/**`: servicio, tipos, handler, assets y sus pruebas.
- Lectura (sin cambios de comportamiento) de `internal/workspace`, `internal/state`, `internal/assets` e `internal/agents`.
- `docs/usage.md` y este documento.

Fuera: el comportamiento de `axiom sync`, el `os.Chdir` de `RunSync` (seguimiento ya anotado) y la TUI.

## Configuración

- **TDD:** off (fuente: `axiom.yaml` no lo configura). Se hacen las comprobaciones funcionales normales.
- **RDD:** off (`axiom review mode status`: decidido por defecto).
- **Ejecutor de pruebas:** `go test` en proceso. **Prohibido compilar binarios sueltos** (Cortex). El JS se prueba con Node, como en los tests existentes.
- **Entrega:** `single-pr` con `size:exception` si supera 400 líneas, y un commit por tarea (preferencia del usuario). Previsión: unas 1.000 líneas con pruebas.

## Tareas

- [x] **T1 · Núcleo del doctor con raíz explícita.**
  - `doctor.Options{Home, Root}` y `cli.BuildDoctorChecks(opts)`.
  - `RunDoctor` resuelve la raíz desde el cwd y delega en el núcleo, de modo que la CLI no cambia.
  - Los checks usan `StateStoreFor(...)` con la raíz dada, sin `os.Getwd`.
  - Etiquetas JSON en `Report`/`Result`.
  - Pruebas con `t.TempDir` y el cwd en otra carpeta.
- [x] **T2 · Endpoint con el resultado real.**
  - `GetDoctorDiagnostics` usa el núcleo con `getRootPath()`.
  - Devuelve `project` y dos secciones (global y proyecto).
  - `Healthy` se calcula de los resultados.
  - Pruebas de contenido, de `Healthy` y de que mira la raíz abierta y no el cwd.
- [x] **T3 · UI por proyecto.**
  - Se pide el doctor al abrir y al cambiar de proyecto.
  - Al cambiar, el informe anterior se invalida y se descartan las respuestas de otro proyecto (clave `project`).
  - Se muestran las secciones global y de proyecto.
  - Pruebas de JS con Node.
- [x] **T4 · Assets y adaptadores del proyecto.**
  - Drift del digest de assets gestionados.
  - Agentes registrados en el estado del proyecto y su presencia en disco (`Adapter.Detect` con la raíz como home de ámbito).
- [x] **T5 · Topología, repos de rol y repo de specs.**
  - `workspace.LocateAt` + `Validate` (errores y avisos como checks).
  - Topología y ruta global en `multirepo`.
  - Cada repo de rol existe y es git.
  - El repo de specs existe y es git.
- [ ] **T6 · Documentación y cierre.**
  - `docs/usage.md` (doctor en el dashboard y comprobaciones de proyecto).
  - PR. Al fusionar, marcar P3 en la auditoría.

## Criterios de aceptación

- Al abrir el dashboard, el doctor muestra las comprobaciones reales del proyecto abierto. Al cambiar de proyecto, el informe anterior desaparece y aparece el del nuevo.
- El resultado no depende del cwd del proceso: hay pruebas con el cwd en otra carpeta.
- `Healthy` es `false` si hay algún fallo.
- Las comprobaciones de proyecto cubren assets, adaptadores, topología, repos de rol y repo de specs, cada una con su remedio.
- `axiom doctor` en la CLI sigue funcionando igual desde la raíz del proyecto.
- `go test ./internal/doctor/... ./internal/cli/ -run Doctor ./internal/dashboard/...` en verde. `internal/cli` completo tarda más de una hora aquí, así que se acota con `-run`.

## Progreso y evidencias

- 2026-11-05: mapa hecho (delegado, solo lectura).
- 2026-11-05: T1 hecha (ruta: escritor delegado). `doctor.Options{Home,Root}`, `Scope` global/proyecto y etiquetas JSON en `Report`/`Result`/`Remedy`, `Report.Failed()`; `cli.BuildDoctorChecks(opts)` sin `os.Getwd` (`RunDoctor` resuelve el cwd una vez y delega; `effectiveStateStoreAt`). Pruebas con cwd en otra carpeta. Observado: `go test ./internal/doctor/... ./internal/dashboard/...` ok; `go test ./internal/cli/ -run 'Doctor|StateStore|InstallState'` ok; `go test ./internal/app/ -run Doctor` ok; `go vet` y `gofmt -l` limpios.
- 2026-11-05: T2 hecha (ruta: escritor delegado). `GetDoctorDiagnostics` ejecuta `cli.BuildDoctorChecks` con `getRootPath()`; devuelve `project`, categoría global/project por chequeo, remedio como recomendación y `Healthy` = sin fallos. Pruebas: lee la raíz abierta y no el cwd, y `Healthy` falso con estado corrupto. Observado: `go test ./internal/doctor/... ./internal/dashboard/...` ok; `go test ./internal/cli/ -run 'Doctor|StateStore|InstallState'` ok; `go test ./internal/app/ -run Doctor` ok; `go vet` y `gofmt -l` limpios.
- 2026-11-05: T3 hecha (ruta: escritor delegado). El doctor se pide al abrir (`loadAllData`) y al cambiar de proyecto: `resetProjectData()` invalida el informe (vuelve a «cargando») y sube una época; las respuestas anteriores al cambio se descartan y las de otro `project` se repiten una vez (la petición GET compartida puede ser la vieja). Ecosistema muestra las secciones global y de proyecto (la categoría `global`/`project` sustituye a agente/herramienta/entorno). Corregido un fallo de `refreshOverview` (`map` pasaba el índice como segundo argumento). Prueba Node nueva (`doctor_project_assets_test.go`); el marcador de `platform_assets_test.go` pasa de `categoryText(` a `doctorSections(`. Sin verificar: aspecto visual. Observado: `go test ./internal/doctor/... ./internal/dashboard/...` ok; `go test ./internal/cli/ -run 'Doctor|StateStore|InstallState'` ok; `go test ./internal/app/ -run Doctor` ok; `node --check` de los JS cambiados, `go vet` y `gofmt -l` limpios.
- 2026-11-05: T4 hecha (ruta: escritor delegado). Nuevos `project:assets` (digest gestionado del binario frente a `ManagedAssetDigest` del estado del proyecto; aviso con remedio `axiom sync --scope workspace`) y `project:adapters` (agentes del estado del proyecto con `Adapter.Detect` usando la raíz como home de ámbito; HOME no cuenta). Solo se añaden si existe `.axiom/state.json`, así que la salida de `axiom doctor` no cambia para quien solo usa la instalación global. Observado: `go test ./internal/doctor/... ./internal/dashboard/...` ok; `go test ./internal/cli/ -run 'Doctor|StateStore|InstallState'` ok; `go test ./internal/app/ -run Doctor` ok; `go vet` y `gofmt -l` limpios.
- 2026-11-05: T5 hecha (ruta: escritor delegado). Nuevos `project:workspace` (`workspace.LocateAt` + `Validate` sin los roles; un axiom.yaml ilegible o con specs ausente es fallo con la causa), `project:topology` (topología, planning home y carpeta maestra en multirepo), `project:role_repos` (falta = fallo, existe sin `.git` = aviso, sin comprobar rama) y `project:specs_repo` (existe y es git; en monorepo-embedded se mira el repo del proyecto). Solo si hay axiom.yaml; sin red ni subprocesos. Deviación: `axiom doctor` ahora lista estas líneas en un proyecto configurado (aditivo); `TestRunDoctor_IntegrationAllMocked` se aísla en un cwd temporal. Observado: `go test ./internal/doctor/... ./internal/dashboard/...` ok; `go test ./internal/cli/ -run 'Doctor|StateStore|InstallState|WorkspaceChecks|ProjectA'` ok; `go test ./internal/app/ -run Doctor` ok; `go vet` y `gofmt -l` limpios.

Plan stop: approved 2026-11-05; implementation continues
