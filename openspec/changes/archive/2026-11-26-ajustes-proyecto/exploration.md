# Exploración: ajustes-proyecto (C-06-e)

Fuente de verdad: `docs/plan/backlog-2026-11/06-ui-human-loop.md` (sección C-06-e) y `docs/plan/backlog-2026-11/README.md` rev. 10 (decisiones P5, P11, P15 y P16).

## Estado actual

- **Librería y lectura.** `gopkg.in/yaml.v3`. Los decodificadores estrictos trabajan sobre `yaml.Node` y los orquesta `internal/workspace/declarations.go:101` (`readDeclarationBlocks`): bloques `technical_context`, `git`, `workflow`, `sdd`, `delivery`, `integrations`, `repositories` y `acceptance_tests`. El resto de claves de primer nivel se ignora (`declarations.go:90`). El `yaml.Node` solo se usa para leer (`loader.go:77`); no hay escritor.
- **Escritura actual.** `axiom.yaml` solo se escribe en el alta: `internal/hub/init.go:247` (`os.WriteFile` de una plantilla de texto). En multirepo el fichero vive en el repo de specs con puntero (`init.go:227,260`); la ruta real sale de `workspace.Locate(...).ConfigPath` (`locate.go:43-50`). `service.go:152` asume `<raíz>/axiom.yaml`.
- **Claves y listas cerradas.**
  - `git`: `gitKeys` = `integration_branch`, `branch_prefix`, `main_branch`, `promotion` (`gitpolicy.go:69`); `promotion` ∈ `branch|release|ask` (`gitpolicy.go:29-31`, `validPromotion` en `:208`); `branch_prefix` con `gitref.ValidPrefix`. `repositories[]`: `path`, `integration_branch`, `branch_prefix`, `promotion` (`gitpolicy.go:70`).
  - `workflow.lane` ∈ `odd|sdd|ask` (`workflow.go:16-21`, validador en `:40`). `checkISOWorkflow` (`declarations.go:129`) rechaza `sdd.iso: true` con `lane: odd` (`ErrISOWorkflowContradiction`, `sdd.go:44`).
  - `sdd`: `sddKeys` = `iso`, `id_pattern`, `explainer_html` (`sdd.go:35`); `explainer_html` ∈ `ask|always|never` (`sdd.go:26-31`); `id_pattern` con `reqid.Parse`.
  - `delivery.split_hint_lines`: entero > 0 (`delivery.go:19`, `:61`).
  - `acceptance_tests`: `enabled` (bool) y `environments` (lista de cadenas) (`acceptance_tests.go:21`); `Validate` lo rechaza activo fuera de multirepo (RF-TOP-19).
  - `workspace` (`types.go:18`): `name`, `topology` (`monorepo-embedded|multirepo`), `specs_repository`, `root`, `skill_roots`; `roles[].gate_policy` ∈ `blocking|deferred|optional` (`types.go:40`; el vocabulario está en `hub/init_validate.go:345`); `governance` (`types.go:44`).
  - Las listas de claves (`gitKeys`, `sddKeys`, `deliveryKeys`, `acceptanceTestsKeys`, `workflowKeys`) y varios validadores son privados. Hay constantes exportadas `Promotion*`, `WorkflowLane*` y `ExplainerHTML*`.
- **Dashboard.** Rutas en `server.go:73-113`; `isLocalRequest` en `server.go:962`. `GET /api/workspace` = `service.go:150` (`GetWorkspace`), con `branchPolicyView` (`:158`, `:204`) que alimenta `branchPolicyRows` (`assets/js/core/branches.js:56`), usada en `screens/ecosystem.js:19,143` y en `screens/projects.js`. Preferencias del hub: `GET|PUT /api/projects/active/worktree-prefs` (RF-WEB-10), DTO `WorktreePrefsDTO` (`types.go:269-286`), proyecto activo por `Service.activeProjectRecord()` (`worktree_prefs.go:64`) y 404 con `projectNotRegisteredError`. **Hueco:** el `PUT` de `worktree_prefs.go:120` no usa `isLocalRequest`.
- **Frontend.** Router por hash (`core/router.js:17`), pantallas importadas en `main.js:15-20`, navegación en `shell.js`, caché del service worker en `assets/sw.js`. Pruebas Go de cadenas JS en `server_test.go:447` y `assets_branch_test.go`.
- **Autofetch (C-05-d).** `hub.LookupAutofetch(projectPath) (enabled, registered bool, err)` (`hub/prefs.go:124`), `hub.SetAutofetch(idOrPath, enabled) (*WorkspaceRecord, bool, error)` (`:142`), `hub.ErrProjectNotRegistered` (`:118`).
- **Specs vivas.** `local-web-dashboard`: último ID RF-WEB-17; RF-WEB-10 («`git:`/`repositories:` en solo lectura; solo preferencias personales») debe modificarse. `workspace-topology`: último ID RF-TOP-20.
- **Restricciones de CI.** Trinquete de código muerto (sin exportados solo para pruebas); canario de secretos (`secrets_canary_test.go:25` `coveredRoutes`, `:41` `excludedRoutes`); `internal/idmigration/residues.yaml` para fixtures con `REQ-`; cadenas esperadas en `server_test.go:447`.

## Enfoques

1. **Paquete nuevo `internal/settings` (catálogo y escritor) más endpoints del dashboard.** El catálogo en Go deriva los valores permitidos de las constantes exportadas de `workspace`. El escritor de *round-trip* `yaml.Node` revalida el documento completo con los decodificadores existentes antes de escribir de forma atómica. La revisión es el SHA-256 del fichero. Esfuerzo: medio. **Elegido.**
2. Catálogo en JS y `PUT` genérico. Duplica listas cerradas y valida poco. Rechazado.

## Decisiones tomadas por el orquestador (delegadas)

- **Editables** (P15: «el resto es editable»): `workspace.name`, `git.*`, `repositories[].{integration_branch,branch_prefix,promotion}` (multirepo), `workflow.lane`, `delivery.split_hint_lines` (P5), `sdd.iso`, `sdd.id_pattern`, `sdd.explainer_html`, `acceptance_tests.{enabled,environments}` (multirepo) y `roles[].gate_policy` (multirepo, con `optional`). `sdd.iso` y `sdd.id_pattern` llevan un efecto descrito explícito.
- **Solo lectura con «abrir en el editor»** (P15): `technical_context`, `workspace.root`, `skill_roots`, `governance.*`. También, por estar fuera de alcance (migrar topologías): `workspace.topology` y `workspace.specs_repository`. `artifact_store` no es editable (persistencia siempre hybrid).
- **Fuera del catálogo:** `integrations.*` (P11, C-07-h), `commit_review`, `docs_publication`, RDD.
- **Monorepo:** no aparecen `roles`, `gate_policy`, `repositories` ni `acceptance_tests`.
- **ISO:** `workflow.lane` no ofrece `odd`, y el validador lo rechaza.
- **Retirar un valor** (`null`) borra la clave y vuelve el valor por defecto del motor; si el bloque queda vacío, se borra.
- **Autofetch (P16):** endpoint propio `GET|PUT /api/settings/personal`, que llama a `hub.LookupAutofetch`/`SetAutofetch`. Las preferencias `worktree_mode` y `worktree_root` siguen en su endpoint, y la pantalla las agrupa en el bloque «Preferencias personales». El `PUT` de `worktree-prefs` pasa a exigir `isLocalRequest`.
- **Sustitución de la política de ramas:** `branchPolicyRows` deja de leerse en Ecosistema y Proyectos; esas pantallas enlazan a «Ajustes del proyecto».

## Riesgos

- El *round-trip* de yaml.v3 puede reformatear sangrado y comillas, o mover comentarios de cabecera y pie. Las anclas, alias y `<<` se rechazan al editar.
- Escritura multirepo: usar `Locate().ConfigPath`.
- Concurrencia: revisión por hash y nueva comprobación justo antes del `rename`.
- Un cambio válido por bloque puede ser inválido en conjunto: validar el documento completo.
