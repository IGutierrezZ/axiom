# Propuesta: Pantalla «Ajustes del proyecto» basada en un catálogo de claves (C-06-e)

## Intención

Hoy `axiom.yaml` solo se escribe en el alta (`internal/hub/init.go:247`, plantilla de texto con `os.WriteFile`) y ninguna ruta HTTP lo edita después: RF-WEB-10 lo prohíbe expresamente. La persona que quiere cambiar el carril (`workflow.lane`), la política de ramas (`git.*`, `repositories[]`), el patrón de ID o el umbral de aviso de tamaño tiene que abrir el YAML a mano, sin validación previa y con riesgo de dejar el proyecto en un estado que los decodificadores estrictos rechazan. Además, el usuario decidió (P16) que el interruptor personal de autofetch se vea y se edite en la UI, y el `PUT` actual de `worktree-prefs` no exige petición local.

El cambio llega ahora porque ya existen todas las claves que debe catalogar (C-02-a, C-02-c, C-03-a, C-06-b, C-06-c) y la preferencia de autofetch (C-05-d), y porque C-07-h depende del catálogo y del escritor que introduce este cambio.

Éxito significa: la persona cambia desde el dashboard las claves de proyecto que gobiernan su flujo, sin abrir el YAML, sin perder comentarios ni orden, con el mismo validador que usa el motor, y sabiendo que el cambio solo afecta a incrementos nuevos.

## Alcance

### Dentro del alcance

- **Paquete nuevo `internal/settings`.**
  - Catálogo de claves: cada entrada declara ruta, etiqueta, descripción, tipo, valores permitidos, si es editable, efecto descrito y topologías en las que aplica. Los valores permitidos se derivan de las constantes exportadas de `internal/workspace` (`Promotion*`, `WorkflowLane*`, `ExplainerHTML*`, vocabulario de `gate_policy`); el catálogo es la única fuente y no duplica las listas cerradas de los decodificadores.
  - Escritor de *round-trip* sobre `yaml.Node`: aplica el parche sobre el árbol, conserva comentarios, orden y bloques ajenos, rechaza editar documentos con anclas, alias o claves `<<`, revalida el documento completo con los decodificadores existentes (`readDeclarationBlocks`, `decodeGit`, `decodeWorkflow`, `checkISOWorkflow`, `workspace.Validate`) y escribe de forma atómica (temporal más `rename`).
  - Revisión SHA-256 del fichero: la escritura exige la revisión leída y la vuelve a comprobar justo antes del `rename`; si no coincide, rechaza sin escribir.
  - Ruta real del fichero mediante `workspace.Locate(...).ConfigPath` (multirepo con puntero incluido).
  - Retirar un valor (`null`) borra la clave y devuelve el valor por defecto del motor; un bloque que queda vacío se borra.
- **Claves editables** (P15, P5): `workspace.name`, `git.{integration_branch,branch_prefix,main_branch,promotion}`, `workflow.lane`, `delivery.split_hint_lines`, `sdd.iso`, `sdd.id_pattern`, `sdd.explainer_html`; solo en multirepo, `repositories[].{integration_branch,branch_prefix,promotion}`, `acceptance_tests.{enabled,environments}` y `roles[].gate_policy` (con `optional`). `sdd.iso` y `sdd.id_pattern` llevan un efecto descrito explícito.
- **Claves en solo lectura con «abrir en el editor»** (P15): `technical_context`, `workspace.root`, `skill_roots`, `governance.*`; además, por quedar fuera de alcance la migración de topologías, `workspace.topology` y `workspace.specs_repository`. `artifact_store` no es editable.
- **Reglas de presentación.** En monorepo no aparecen `roles`, `gate_policy`, `repositories` ni `acceptance_tests`. En un proyecto ISO, `workflow.lane` no ofrece `odd` y el validador lo rechaza.
- **Endpoints del dashboard.**
  - `GET|PUT /api/settings`: catálogo con valores actuales y revisión; escritura con revisión esperada y errores tipados en castellano.
  - `GET|PUT /api/settings/personal`: autofetch del proyecto activo mediante `hub.LookupAutofetch` y `hub.SetAutofetch`, con 404 de proyecto no registrado.
  - Todas las escrituras exigen `isLocalRequest`, incluido el `PUT /api/projects/active/worktree-prefs` existente.
  - `GET /api/workspace` sigue siendo de solo lectura (405 para el resto de métodos).
- **Pantalla JS «Ajustes del proyecto»**: generada desde el catálogo, con el aviso «Editar solo afecta a cambios nuevos» y el bloque «Preferencias personales» (autofetch, `worktree_mode` y `worktree_root`, estos dos sobre su endpoint actual). Ruta por *hash*, entrada de navegación y caché del *service worker*.
- **Retirada de `branchPolicyRows`** de Ecosistema y Proyectos: esas pantallas enlazan a «Ajustes del proyecto».
- **Canario de secretos**: las rutas nuevas se registran en `coveredRoutes` o `excludedRoutes`.
- **Documentación**: manual del dashboard y `axiom.example.yaml` (nota de que las claves se pueden editar desde la UI), si procede.

### Fuera del alcance

- `integrations.*`: lo edita C-07-h con su propia pantalla, usando el catálogo y el escritor de este cambio (P11).
- Cualquier control de RDD (global, por proyecto o por clon).
- Preferencias del instalador en `state.json` (persona, preset, modelos por fase).
- `openspec/config.yaml`.
- Editar secretos o credenciales.
- Migrar `axiom.yaml` entre topologías.
- `commit_review` (campo de la decisión de la puerta `tasks`) y `docs_publication` (campo de `kickoff.yaml`): no son claves de `axiom.yaml`.
- Modificar instantáneas de `kickoff.yaml` de cambios ya sellados.

## Capacidades

### Capacidades nuevas

Ninguna.

### Capacidades modificadas

- `local-web-dashboard`:
  - MODIFIED RF-WEB-10: deja de prohibir toda escritura de `axiom.yaml`; las escrituras pasan por `/api/settings` con el catálogo, el validador completo y la revisión; `/api/workspace` sigue en solo lectura; las preferencias personales incluyen el autofetch y toda escritura exige petición local.
  - MODIFIED RF-WEB-07 (probable): la navegación incorpora «Ajustes del proyecto» y la vista de topología deja de mostrar la política de ramas.
  - ADDED a partir de RF-WEB-18: pantalla «Ajustes del proyecto» generada desde el catálogo (reglas por topología e ISO, aviso «solo cambios nuevos», solo lectura con «abrir en el editor», ningún control de RDD); endpoint `GET|PUT /api/settings` (revisión, rechazo sin escritura); endpoint `GET|PUT /api/settings/personal`.
- `workspace-topology`:
  - ADDED a partir de RF-TOP-21 (si la fase de especificación lo confirma): catálogo de claves editables como fuente única derivada del esquema, y escritor de *round-trip* que conserva comentarios y orden, revalida el documento completo, escribe de forma atómica con revisión y rechaza anclas, alias y `<<`.

`multi-project-hub` no cambia de requisito: el dashboard reutiliza `LookupAutofetch` y `SetAutofetch` (RF-HUB-07, RF-HUB-13). La fase de especificación debe confirmarlo.

## Enfoque

Enfoque 1 de la exploración (elegido): paquete `internal/settings` con catálogo en Go y escritor `yaml.Node`, más endpoints finos en `internal/dashboard`. El enfoque 2 (catálogo en JS y `PUT` genérico) se descarta porque duplica listas cerradas y valida poco.

1. `internal/settings` depende de `internal/workspace` y nunca al revés. Si algún validador o lista privada hace falta, se exporta solo cuando lo use código de producción (trinquete de código muerto).
2. Flujo de escritura: leer el fichero, calcular la revisión, comprobar la revisión esperada, aplicar el parche al `yaml.Node`, serializar, revalidar el documento serializado con los decodificadores estrictos, comprobar de nuevo la revisión y hacer el `rename`. Cualquier fallo devuelve el error del validador sin tocar el fichero.
3. El servicio del dashboard expone el catálogo ya filtrado por topología e ISO; la pantalla JS lo pinta sin conocer las claves.
4. La instantánea del kickoff ya manda sobre `axiom.yaml` en los cambios sellados; la pantalla solo lo comunica, sin tocar ningún `kickoff.yaml`.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/settings/` | Nuevo | Catálogo, escritor `yaml.Node`, revisión SHA-256, escritura atómica |
| `internal/workspace/` (`types.go`, `declarations.go`, `gitpolicy.go`, `workflow.go`, `sdd.go`, `delivery.go`, `acceptance_tests.go`) | Modificado (mínimo) | Exportar lo que el catálogo necesite, sin duplicar listas |
| `internal/dashboard/server.go` | Modificado | Rutas `/api/settings` y `/api/settings/personal`; `isLocalRequest` en las escrituras, también en `worktree-prefs` |
| `internal/dashboard/service.go`, `internal/dashboard/worktree_prefs.go`, `internal/dashboard/types.go` | Modificado | Servicio y DTO de ajustes; guarda local en el `PUT` existente |
| `internal/dashboard/secrets_canary_test.go` | Modificado | Registro de las rutas nuevas |
| `internal/dashboard/assets/js/screens/` (pantalla nueva, `ecosystem.js`, `projects.js`), `core/branches.js`, `core/router.js`, `main.js`, `shell.js`, `assets/sw.js` | Nuevo/Modificado | Pantalla «Ajustes del proyecto»; retirada de `branchPolicyRows` |
| `internal/dashboard/server_test.go` (`:447`), `assets_branch_test.go` | Modificado | Cadenas JS esperadas |
| `internal/idmigration/residues.yaml` | Modificado (si procede) | Fixtures con `REQ-` del patrón de ID |
| `axiom.example.yaml`, documentación del dashboard | Modificado | Nota de edición desde la UI |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| yaml.v3 reformatea sangrado o comillas, o mueve comentarios de cabecera y pie | Media | Golden de un `axiom.yaml` comentado; sangrado fijado en el codificador; rechazo explícito de anclas, alias y `<<` |
| Un cambio válido por bloque resulta inválido en conjunto | Media | Revalidar siempre el documento completo serializado antes de escribir |
| Escrituras concurrentes (UI y editor) pisan cambios | Media | Revisión SHA-256 esperada y nueva comprobación justo antes del `rename` |
| Escribir en la ruta equivocada en multirepo | Baja | Usar solo `Locate().ConfigPath` |
| Escritura remota desde otra máquina | Baja | `isLocalRequest` en todas las escrituras, incluida `worktree-prefs` |
| Fugas de secretos por las rutas nuevas | Baja | Canario de secretos; `integrations.*` fuera del catálogo |
| Exportados solo para pruebas o código muerto en CI | Media | Exportar solo lo que use producción; pruebas en el mismo paquete |
| Pruebas inestables por puertos compartidos | Baja | Sin `t.Parallel` con `freePort` ni con el hub compartido |
| Mecanismo de «abrir en el editor» sin definir | Media | Decisión abierta para el orquestador (ver Dependencias) |

## Plan de reversión

- Revertir la PR única (*merge commit*): desaparecen las rutas `/api/settings*`, la pantalla y el paquete `internal/settings`; RF-WEB-10 vuelve a su redacción anterior al no integrarse el delta.
- `axiom.yaml` está versionado: cualquier escritura de la UI se deshace con `git checkout`/`git revert` del fichero. El escritor nunca deja un fichero a medias (escritura atómica) ni un documento inválido (revalidación previa).
- Los `kickoff.yaml` sellados no se tocan, así que los cambios en curso no necesitan reversión.
- La guarda `isLocalRequest` del `PUT` de `worktree-prefs` puede revertirse de forma aislada si rompe algún cliente legítimo.

## Dependencias

- Ya integradas según la exploración: C-02-a (`git.main_branch`, `git.promotion`), C-02-c (`delivery.split_hint_lines`), C-03-a (`sdd:` e `id_pattern`), C-06-b (`explainer_html`), C-06-c (`acceptance_tests`), C-01-b (roles solo en multirepo), C-01-c (sin RDD en el dashboard), C-05-d (`hub.LookupAutofetch`/`SetAutofetch`).
- Bloquea a C-07-h, que reutiliza el catálogo y el escritor.
- **Decisión abierta (para el orquestador):** qué hace «abrir en el editor» en las claves de solo lectura. Recomendación: mostrar la ruta absoluta de `axiom.yaml` con botón de copiar, sin lanzar procesos desde el servidor. No bloquea la especificación del resto.

## Restricciones de CI

- Trinquete de código muerto: ningún exportado solo para pruebas.
- Canario de secretos con las rutas nuevas registradas.
- `internal/idmigration/residues.yaml` para fixtures con `REQ-`.
- Cadenas JS esperadas en `internal/dashboard/server_test.go:447`.
- Pruebas localizadas con `-run` sobre los paquetes tocados; la batería completa, en CI.
- Nada de `t.Parallel` con `freePort` ni con el hub compartido.
- No compilar binarios sueltos: arneses solo en proceso con `go test`.

## Criterios de éxito

- [ ] Cambiar `workflow.lane` desde la UI escribe `axiom.yaml` conservando comentarios y orden (prueba con golden de un `axiom.yaml` comentado).
- [ ] Un valor inválido se rechaza con el error del validador y el fichero no cambia (misma revisión).
- [ ] Una revisión desactualizada se rechaza sin escribir.
- [ ] `odd` en un proyecto ISO no se ofrece y, si se envía, se rechaza.
- [ ] En monorepo no aparecen `roles`, `gate_policy`, `repositories` ni `acceptance_tests`.
- [ ] La pantalla muestra «solo afecta a cambios nuevos» y un cambio ya sellado conserva su instantánea de `kickoff.yaml`.
- [ ] No aparece ningún control de RDD ni de `integrations.*`.
- [ ] El autofetch es editable desde «Preferencias personales» y persiste en `~/.axiom/workspaces.json`, sin tocar ficheros del proyecto.
- [ ] Las escrituras desde una petición no local se rechazan, también en `worktree-prefs`.
- [ ] Ecosistema y Proyectos ya no usan `branchPolicyRows` y enlazan a «Ajustes del proyecto».
- [ ] Las claves de solo lectura (`technical_context`, `workspace.root`, `skill_roots`, `governance.*`, `workspace.topology`, `workspace.specs_repository`) se muestran sin control de edición.
