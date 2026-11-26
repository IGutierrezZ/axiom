# Propuesta: plantillas de nombre de rama y ramas de código diferidas a la entrada del rol (C-07-f)

## Intención

Hoy el nombre de cada rama lo fija el código: `<prefijo>/<cambio>` para la rama del cambio y `<rama del cambio>-<rol>` para la de rol. Solo el prefijo se configura. Los equipos que trabajan con un gestor de trabajo tienen sus propias convenciones. Por ejemplo, en KVP la rama de back es `feature/<taskId>` y la de front es `feature/TASK-<taskId>`. Con Axiom no pueden seguirlas: o renuncian a su convención o recurren a `--branch` en cada cambio, que solo cubre la rama del cambio.

Además, en `multirepo` el primer `ensure` posterior al sellado crea todas las ramas de código y de rol a la vez. Ocurre aunque el rol no haya empezado y aunque su Task todavía no exista. Así, una convención basada en la Task del rol no se puede aplicar: cuando nace la rama, el identificador aún no existe.

El cambio sirve para que cada proyecto, repositorio y rol declare cómo se llama su rama y pueda usar las referencias externas del work item y de la Task. También hace que las ramas cuyo nombre depende de la Task del rol se creen cuando ese rol entra (`ensure --role`), no al arrancar el cambio. Sin plantilla declarada, todo se comporta exactamente igual que hoy.

**Por qué ahora:** ya existen las dos dependencias, el vínculo `work_item` de `change start` (C-07-d, origen de `{ref}`) y los vínculos `task` por rol con la barrera `task_not_bound` (C-07-e, RF-BRA-33). La sección 02 (promoción, `main_branch`, RF-BRA-28) ya está integrada en el mismo bloque `git:` y en la misma instantánea, así que el solape que señalaba el plan ya está resuelto.

## Decisiones tomadas por el orquestador (delegadas de diseño)

Se registran como decisiones ya tomadas. La propuesta no las reabre.

- **D1 (alcance).** El diferido por rol (opción B del usuario) entra en este cambio. Una rama cuyo nombre exige `{task_ref}` no se crea al arrancar ni en el primer `ensure` sin rol. Se crea en el `ensure --role <rol>` de ese rol, después de la barrera `task_not_bound` de C-07-e. Una rama cuya plantilla no usa `{task_ref}` se crea en el mismo momento que hoy. Sin plantilla, nada cambia.
- **D2 (congelado).** El kickoff es de escritura única (`kickoff/seal.go`, `PublishNoReplace`), así que no se puede reescribir en el `ensure` del rol. Por eso la instantánea guarda la **plantilla**, en un campo aditivo de `GitSnapshotBranch`, y el **nombre** resuelto se congela en el registro local de ramas (`branch.Record`, RF-BRA-15) en el primer `ensure` que crea la rama. Las ejecuciones posteriores leen el nombre del registro.
- **D3 (forma YAML).** `branch_template` es un escalar en `git:`, en cada entrada de `repositories:` y en `roles.<id>:`. La precedencia es rol > repositorio > `git:`. Si ningún nivel declara plantilla, los nombres son los de hoy.
- **D4 (variables).**

  | Variable | Valor |
  |---|---|
  | `{prefix}` | `branch_prefix` efectivo, incluido el que fija `--prefix`. |
  | `{kind}` | `fix` si al resolver se conoce que el cambio tiene perfil bug (staging o vínculo de `change start` con perfil bug); si no, `feature`. |
  | `{change}` | Nombre del cambio. |
  | `{slug}` | Nombre del cambio sin el prefijo del ID de work item (`US-2525-alta-de-usuarios` → `alta-de-usuarios`); igual a `{change}` si no hay work item. |
  | `{role}` | Identificador del rol. |
  | `{ref}` | ID del vínculo `work_item` (p. ej. `2525`), leído de `integrations.yaml` o, si el vínculo aún no existe, del staging de `change start`. |
  | `{task_ref}` | ID de la Task del rol (`TaskLinks`). |

  Reglas:
  - Una variable desconocida o una llave mal formada se rechaza con un error tipado al cargar `axiom.yaml`.
  - `{role}` y `{task_ref}` solo son válidas en plantillas que nombran ramas de rol o ramas de repositorios de código atribuibles a un rol.
  - La rama que lleva `openspec/` no puede usar `{role}` ni `{task_ref}`: se rechaza al cargar. Esa rama es la del cambio en un monorepo y la del repositorio de specs en un multirepo.
  - Si una variable no tiene valor al resolver (`{ref}` sin work item, `{task_ref}` sin Task), el `ensure` se rechaza con un error tipado que incluye la continuación.
  - El nombre resultante se valida con `gitref`. Si no es válido, se rechaza con un error tipado.
- **D5 (compatibilidad).** Un cambio ya sellado sin plantilla en su instantánea conserva el comportamiento actual y no se migra.
- **D6 (catálogo C-06-e).** Se añaden tres claves `TypeString` en `internal/settings/catalog.go`: `git.branch_template`, `repositories[].branch_template` y `roles[].branch_template`.

**Corrección justificada del criterio de terminado del plan** (`docs/plan/backlog-2026-11/07-integraciones.md`, C-07-f). El plan dice «el nombre queda congelado en la instantánea tras el primer `ensure`». Pasa a decir «la plantilla queda en la instantánea y el nombre queda congelado en el registro local en el primer `ensure` que crea la rama». El criterio original no se puede implementar: la instantánea es de escritura única (RF-KOF-06) y ninguna orden, incluida `ensure`, puede modificarla.

## Alcance

### Dentro del alcance

1. **Esquema de `axiom.yaml`.** La clave `branch_template` entra en `git:` (RF-TOP-09), en las entradas de `repositories:` (RF-TOP-10) y en `roles.<id>:`. Al cargar se valida:
   - que todas las variables son conocidas y las llaves están bien formadas;
   - que la rama con `openspec/` no usa `{role}` ni `{task_ref}`;
   - que el nombre, renderizado con valores de muestra, es un nombre de referencia git válido.

   Los mensajes que enumeran las claves admitidas de `git:` y de `repositories:` incluyen `branch_template`.
2. **Motor de plantillas puro** en `internal/branch`, sin E/S, con dos funciones: renderizar y listar las variables. Cada valor sustituido se valida como componente de rama (`gitref.ValidBranchComponent`). La excepción es `{prefix}`, que admite `/`. Si un valor no es seguro para git, el `ensure` se rechaza con un error tipado; el valor nunca se sanea en silencio.
3. **Resolución de variables en el motor de `branch`.** El motor obtiene cada valor de su fuente y los entrega al planificador:
   - `{ref}`: del vínculo `work_item` de `integrations.yaml` o, si todavía no existe, del staging de `change start`;
   - `{task_ref}`: de los vínculos `task` por rol;
   - `{kind}`: del perfil de `change start`;
   - `{slug}`: del nombre del cambio y del ID del work item.

   Si `integrations.yaml` no se puede leer y hay un plugin activo, el motor falla cerrado, igual que la barrera de RF-BRA-33.
4. **Planificador.** La precedencia de nombres pasa a ser: excepción por cambio (`--branch`) > nombre de la instantánea > nombre del registro > plantilla (instantánea o política) > nombre actual. La plantilla efectiva sigue la precedencia rol > repositorio > `git:`. Sin plantilla, el planificador sigue el camino de código actual, de modo que los nombres son idénticos byte a byte. La rama de archivado se sigue derivando de la rama del cambio ya resuelta (`<rama del cambio>-archive`).
5. **Instantánea del kickoff (RF-KOF-06).** El seal añade un campo aditivo con la plantilla (`omitempty`) a cada rama de código. Si la plantilla usa `{task_ref}`, no escribe el nombre. La rama con `openspec/` y las ramas sin `{task_ref}` siguen llevando su nombre resuelto, como hoy. Un binario anterior ignora el campo nuevo.
6. **Congelado en el registro (RF-BRA-15).** El primer `ensure` que resuelve el nombre de una rama con plantilla lo guarda en el registro. Las ejecuciones posteriores leen ese nombre y no vuelven a renderizar la plantilla. Si el registro se reconstruye, el nombre se recupera de las ramas existentes, la instantánea y la plantilla, con las mismas entradas versionadas.
7. **Diferido por rol (D1).**
   - `EnsureRequest` recibe el rol de `--role`.
   - El planificador marca como pendiente la rama cuya plantilla usa `{task_ref}` cuando el rol que la reclama no es el que entra.
   - Una rama pendiente no se crea ni se nombra.
   - `ensure` y `status` la listan como pendiente en un campo aditivo del JSON.
   - RF-BRA-33 cambia: con `--role`, el plan puede diferir del de la invocación sin bandera, pero solo en las ramas de ese rol que estaban pendientes.
8. **Guarda (`verdict.go`).** El nombre esperado de cada rama sale de la instantánea o del registro, no de recomponer `prefijo + "/" + cambio`. Así, una rama con plantilla no se marca como inesperada.
9. **Carril ODD (RF-BRA-14, «misma política»).** `git.branch_template` también se aplica a la rama del cambio ODD, con su nombre congelado en el registro. En ODD, `{role}` y `{task_ref}` se rechazan con un error tipado.
10. **Catálogo de ajustes (D6).** Se registran las tres claves. Siguiendo la regla de topología vigente de RF-TOP-21, `repositories[]` y `roles[]` solo se muestran en `multirepo`. La validación de la plantilla también se aplica al escribir desde `PUT /api/settings`, a través del revalidado completo del escritor (RF-TOP-22) o de una llamada explícita al validador; el diseño elige cuál.
11. **Documentación y ejemplo.** `axiom.example.yaml`: líneas comentadas en `git:` y `repositories:` con un ejemplo por nivel. Esas líneas se ejecutan en `TestExampleGitBlockIsValidOnceUncommented`. También se documenta el bloque `git:` en la documentación de usuario, si existe.
12. **Códigos de rechazo nuevos**, cada uno con `Continuation`, de alta en `types.go` y `labels.go` y cubierto por el trinquete de resolución de rechazos. Son, como mínimo: plantilla inválida al cargar (a través de `config_invalid`), variable sin valor al resolver y nombre renderizado inválido (a través de `invalid_name`). Los nombres exactos los fija el diseño.

### Fuera del alcance

- La base y la promoción de la rama (sección 02, RF-BRA-28/29): la plantilla solo decide el **nombre**.
- Cambiar el nombre de la carpeta del worktree. Sigue usando el nombre del cambio, no el de la rama, así que el límite de longitud de RF-BRA-11 no cambia.
- Migrar cambios ya sellados (D5) o renombrar ramas existentes al cambiar la plantilla. Editar la plantilla solo afecta a cambios nuevos y a ramas todavía no creadas.
- Una bandera `--kind` en `ensure`. `{kind}` sale del perfil de `change start`. Sin plugin vale `feature`, y `--prefix`/`--branch` siguen disponibles como excepción.
- Leer `{kind}` de la declaración de bug de `proposal.md`, que no existe al crear la rama con `openspec/`.
- Plantillas condicionales o mapas `branch_templates` bajo `git:` (descartados en la exploración, enfoque A2/A3).
- Crear Tasks o ramas al aprobar el design. Al aprobar el design solo se crean Tasks (C-07-e).

## Capacidades

### Capacidades nuevas

Ninguna.

### Capacidades modificadas

- `change-branch-lifecycle`:
  - RF-BRA-03 (resolución de la política: entra la plantilla con su precedencia y los rechazos tipados).
  - RF-BRA-04 (modelo de ramas: nombre de la rama de rol y de la rama de código con plantilla).
  - RF-BRA-15 (el registro congela el nombre resuelto).
  - RF-BRA-14 (ODD con plantilla, sin variables de rol).
  - RF-BRA-16 (`status` lista las ramas pendientes).
  - RF-BRA-19 (veredicto con el nombre registrado).
  - RF-BRA-33 (`--role` llega al motor y crea las ramas diferidas de ese rol).
  - Requisitos nuevos `prov:BRA:1..n`, con ID provisional: motor y variables de plantilla; diferido por rol de las ramas con `{task_ref}`; rechazos tipados de resolución.
- `sdd-lane-kickoff`: RF-KOF-06 (la instantánea guarda la plantilla de cada rama de código y omite el nombre de las ramas que dependen de `{task_ref}`; campo aditivo que un binario anterior ignora).
- `workspace-topology`:
  - RF-TOP-09 (`git.branch_template`).
  - RF-TOP-10 (`repositories[].branch_template` y la lista de claves admitidas).
  - Entrada de roles de `axiom.yaml`, `roles.<id>.branch_template` (RF-TOP-03/06 según dónde la spec describe `roles:`; lo fija `sdd-spec`).
  - RF-TOP-21 (enumera las claves editables del catálogo: hay que añadir las tres).
- `local-web-dashboard`: **sin delta previsto**. RF-WEB-18 genera la pantalla desde el catálogo y no enumera las claves de forma exhaustiva (`repositories[]` en bloque y `roles[].gate_policy` como ejemplo con vocabulario), y RF-WEB-19 delega en RF-TOP-21. Si `sdd-spec` encuentra una enumeración cerrada que lo contradiga, añadirá el delta.

## Enfoque

Se sigue la recomendación de la exploración (A1 + B1 + C1 + D1):

- **A1.** Clave escalar en los tres niveles, con un caso nuevo en `branchValidator` y una línea más en `gitKeys` y `repositoryKeys`. Los roles no pasan por `decodeStringKeys`, así que su validación se añade a mano en la carga.
- **B1.** Función pura en `internal/branch` (p. ej. `template.go`). El motor (`branch/engine`) resuelve las variables, porque solo él lee `integrations.yaml` y el staging, y las entrega en `PlanInput`. Así `Snapshot` (seal) y `ensure` comparten el mismo cálculo y nunca discrepan (`snapshot.go:22`).
- **C1.** Plantilla en la instantánea y nombre en el registro. `planRepository` aprende a leer el registro antes que la plantilla.
- **D1.** `EnsureRequest.Role` y ramas «pendientes» en el plan. Un solo `ensure`, idempotente.

**Coherencia entre clones (para el diseño).** El registro es local a cada clon. Para que otra persona, con `ensure --role` en otro clon, obtenga el mismo nombre (y adopte la rama publicada, RF-BRA-27), todas las entradas de la plantilla deben estar versionadas:

- `{task_ref}` y `{ref}` lo están, en `integrations.yaml`.
- `{kind}` no lo está: el perfil solo vive en el staging y en `context/work-item.json`, que son locales.

Se propone que el seal, con el staging aún presente porque `--adopt` va después del seal, congele en la instantánea los valores de `{kind}` y `{ref}` conocidos en ese momento. Serían campos aditivos y `omitempty`. Son entradas de política, no resultados, así que encaja con RF-KOF-06. Es un refinamiento de D2 que el diseño confirma o descarta. Si se descarta, el riesgo de la tabla queda abierto.

**Rama de seguimiento de un repositorio con dos o más roles (para el diseño).** Esa rama no es atribuible a un único rol, así que su plantilla efectiva es la del repositorio o la de `git:`. Si esa plantilla usa `{role}` o `{task_ref}`, se propone un rechazo tipado al planificar, en el seal o en el `ensure`, que nombre el repositorio y la clave `repositories[].branch_template` como continuación. No se aplica ninguna alternativa silenciosa.

**Monorepo.** La única rama de un monorepo es la del cambio, que lleva `openspec/`. Por tanto, `git.branch_template` con `{role}` o `{task_ref}` se rechaza al cargar. Para `roles.<id>.branch_template` en `monorepo-embedded` se propone una advertencia no bloqueante de «sin efecto», como el precedente de `promotion` sin `main_branch` en RF-TOP-09.

## Zonas afectadas

| Zona | Impacto | Descripción |
|------|---------|-------------|
| `internal/workspace/gitpolicy.go`, `types.go` | Modificado | Clave `branch_template` en `git:`, `repositories:` y `RoleConfig`; validación de la plantilla al cargar |
| `internal/branch/template.go` | Nuevo | Motor puro de plantillas (renderizar y listar variables) |
| `internal/branch/policy.go`, `planner.go`, `planner_snapshot.go`, `naming.go` | Modificado | Precedencia de nombres, plantilla efectiva por rol/repositorio/`git:`, ramas pendientes |
| `internal/branch/types.go`, `labels.go` | Modificado | Códigos de rechazo nuevos con continuación; campo de rama pendiente |
| `internal/branch/engine/engine.go`, `ensure.go`, `snapshot.go`, `verdict.go`, `archive_plan.go` | Modificado | Resolución de variables, rol en el motor, congelado en el registro, guarda con el nombre registrado |
| `internal/kickoff/gitsnapshot.go` | Modificado | Campo aditivo de plantilla (y, si el diseño lo confirma, valores de `kind`/`ref`) |
| `internal/cli/branch.go`, `branch_role.go` | Modificado | Pasar `Role` al motor; JSON aditivo de ramas pendientes |
| `internal/settings/catalog.go` (+ `patch.go` si hace falta) | Modificado | Tres filas `TypeString` |
| `axiom.example.yaml` | Modificado | Ejemplos comentados en `git:` y `repositories:` |
| `openspec/changes/plantilla-ramas/specs/…` | Nuevo | Deltas de `change-branch-lifecycle`, `sdd-lane-kickoff` y `workspace-topology` |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| Regresión de nombres sin plantilla (todo el producto depende de `<prefijo>/<cambio>`) | Media | Camino `template == ""` idéntico al actual; pruebas de igualdad byte a byte de los nombres existentes en `planner_test.go`, `snapshot_test.go` y `ensure_multirepo_test.go` |
| Orden `ensure` → seal → `--adopt`: en el primer `ensure`, `{ref}` solo está en el staging | Media | Leer del staging en el git common dir; sin staging ni vínculo, rechazo tipado (nunca un valor vacío) |
| Nombre distinto entre clones porque `{kind}` no está versionado | Media | Congelar `kind`/`ref` en la instantánea al sellar (pendiente de confirmar en el diseño); si no, documentar la limitación |
| La guarda recompone nombres en tres sitios y marcaría ramas con plantilla como inesperadas | Alta si se omite | Leer el nombre de la instantánea o del registro en `expectedNames`; pruebas de veredicto con plantilla |
| Cambio de contrato de RF-BRA-33 («con `--role` el plan no difiere») | Media | Solo difiere con plantillas que usan `{task_ref}`; sin ellas, la invariancia se mantiene y se prueba |
| Un binario anterior rechaza la clave desconocida en `git:` o `repositories:` | Baja | Mismo precedente que `promotion` (`TestGitBlockIsToleratedByThePreviousBinary`); campo de la instantánea `omitempty` |
| `{task_ref}` de otros plugins con caracteres no válidos para git | Baja | Validación por componente y rechazo tipado, sin saneado |
| Ampliar la plantilla al carril ODD (RF-BRA-14) no estaba explícito en C-07-f | Baja | Sin plantilla no hay efecto; si el orquestador lo prefiere, se acota a SDD y RF-BRA-14 recibe una excepción explícita |
| Deadcode ratchet y trinquete de rechazos | Baja | Toda función exportada alcanzable desde la carga o el plan; toda negativa con `Continuation` |

## Plan de reversión

- Cada pieza es aditiva. Revertir los commits del cambio en la rama `inc/plantilla-ramas` (o la PR, si ya se fusionó) devuelve los nombres actuales.
- Los cambios sellados con el binario nuevo conservan `kickoff.yaml` con el campo `template`, que el binario anterior ignora.
- Un cambio cuyas ramas diferidas aún no se crearon vuelve a crearlas todas en el primer `ensure`, con el nombre actual.
- Las ramas ya creadas con nombre de plantilla siguen existiendo. El registro local guarda sus nombres, pero el planificador anterior no lo lee para el nombre: esos cambios se completan con `--branch` o se recrean. Por eso se recomienda no fusionar a `main` ningún cambio de producto que dependa de una plantilla hasta comprobar este cambio en uso.
- Basta con retirar `branch_template` de `axiom.yaml` para que los cambios nuevos vuelvan al comportamiento actual sin revertir código.

## Dependencias

- C-07-d (`change start`, vínculo `work_item`, staging): integrado.
- C-07-e (vínculos `task` por rol, barrera `task_not_bound`, RF-BRA-33/41): integrado.
- Sección 02 (promoción, `main_branch` en la instantánea, RF-BRA-28): integrada. Solapa en `gitpolicy.go` y `gitsnapshot.go`, pero ya no hay conflicto.

## Criterios de éxito

- [ ] Sin `branch_template` en ningún nivel, los nombres de la rama del cambio, de las ramas de rol y de la rama de archivado, la instantánea y el JSON de `ensure` son idénticos a los actuales (las pruebas existentes pasan sin cambios).
- [ ] Un `multirepo` con `repositories[].branch_template: "feature/TASK-{task_ref}"` y la Task `1235` del rol produce `feature/TASK-1235` en el `ensure --role` de ese rol.
- [ ] Antes de ese `ensure --role`, la rama no existe y `ensure` y `status` la listan como pendiente.
- [ ] La instantánea guarda la plantilla de esa rama y no su nombre. El nombre queda en el registro local tras el primer `ensure` que la crea, y un `ensure` posterior lo lee de ahí aunque cambie la plantilla de `axiom.yaml`.
- [ ] La rama con `openspec/` con `git.branch_template: "{prefix}/{ref}-{slug}"` y el work item `2525` se llama `<prefijo>/2525-alta-de-usuarios` desde el primer `ensure`, antes de `--adopt`.
- [ ] `{role}` o `{task_ref}` en la plantilla de la rama con `openspec/`, una variable desconocida o una llave mal formada se rechazan al cargar `axiom.yaml` con un error tipado que nombra la clave.
- [ ] Una variable sin valor al resolver o un nombre renderizado que no es una referencia git válida se rechazan en el `ensure` con un código tipado y una continuación, sin crear nada.
- [ ] Un cambio sellado antes de esta capacidad, sin plantilla en la instantánea, conserva sus nombres actuales.
- [ ] Las tres claves aparecen en el catálogo de ajustes (`repositories[]` y `roles[]` solo en `multirepo`), y un valor inválido enviado por `PUT /api/settings` se rechaza.
- [ ] Pasan las pruebas focalizadas de `internal/workspace`, `internal/branch/...`, `internal/kickoff`, `internal/settings` e `internal/cli` (las de rama y kickoff), el trinquete de rechazos y el deadcode ratchet.

## Estimación de tamaño

**L**, no la M del plan. Son unas 1.300-1.900 líneas cambiadas con pruebas (alrededor del 50 % en pruebas). La M de 500-800 líneas no incluía el diferido por rol (D1), el rol en el motor, la lectura del registro en el plan ni la guarda.

Se prevén cuatro unidades de trabajo, entregadas en una sola PR (`single-pr`):

1. Esquema, validación de la plantilla, catálogo y ejemplo.
2. Motor de plantillas, resolución de variables, planificador e instantánea.
3. Congelado en el registro y guarda.
4. Diferido por rol: `EnsureRequest.Role`, ramas pendientes y JSON y `status` aditivos.
