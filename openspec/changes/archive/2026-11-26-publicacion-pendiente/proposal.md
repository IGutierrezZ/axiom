# Proposal: Estado «aprobada sin publicar» en `axiom sdd status` (C-05-c)

`axiom sdd status --json` proyectará `publication: pending` con la orden exacta de `axiom sdd publish` cuando un cambio tenga una aprobación (o un momento `auto` cumplido) cuyos documentos no hayan llegado a `origin`. La doctrina del orquestador ordenará ejecutar esa orden una vez antes de la fase siguiente y al reanudar. Así una publicación olvidada deja de pasar desapercibida.

## Intent

**Problema.** `axiom sdd gate record` registra la aprobación y devuelve la orden de publicación sin ejecutarla (REQ-26.5, `internal/cli/sdd_gate_handoff.go:171-192`). Esa orden la ejecuta el orquestador por instrucción. La publicación se pierde sin que nadie lo detecte cuando:

- el orquestador la olvida;
- la sesión se corta o el contexto se compacta;
- el push falla.

La aprobación queda en `gates.yaml`, pero el equipo no ve los documentos en `origin/<rama>`.

**Por qué ahora.** C-05-b ya está fusionado: `continuous` con `docs_publication: auto` publica en cuatro momentos (spec, design, tasks y archivo; REQ-27.13). Hay más momentos de publicación y, con ellos, más ocasiones de perder uno. El objetivo del usuario es que la documentación quede siempre subida en cada parada aprobada (`docs/plan/backlog-2026-11/05-conocimiento-equipo.md`, C-05-c).

**Éxito.** El status nativo detecta la publicación pendiente sin red y la proyecta con la invocación exacta. El orquestador la resuelve antes de lanzar la fase siguiente o al reanudar. Una negativa de `publish` sigue siendo informativa.

## Scope

### In Scope

1. **Cálculo de solo lectura de los hechos de publicación** (motor de ramas). A partir de `PublicationTarget`, se calcula, sin red y con referencias locales:
   - rutas de documentación del cambio con cambios sin confirmar (lista cerrada de REQ-27.5, sin el estado local con punto);
   - commits de la rama sin subir, con el mismo rango que `publish` (REQ-27.8): excluye `refs/remotes/origin/<rama>` y `refs/remotes/origin/<integración>`;
   - si existe `refs/remotes/origin/<rama>`. Si no existe, es la primera publicación y cuenta como pendiente, no como error;
   - si algún commit sin subir toca rutas que no son documentación (código). Solo se calcula si se adopta la opción B de la decisión D1.
2. **Proyección en `axiom sdd status --json`**, junto a la de compuertas en `governance`: estado, motivos, remoto, rama e invocación. La invocación es la misma `branch.PublishInvocation(...)` que lleva `publication.invocation` de la pregunta de compuerta. Para evitar deriva, se reutiliza `resolveGatePublication`.
3. **Condición de proyección.** Se cumplen (a) y (b):
   - (a) existe una aprobación registrada de una compuerta distinta de `integration` (`checkpointed`), o se ha cumplido un momento de REQ-27.13 en `continuous` con `docsPublication: auto`;
   - (b) hay documentación sin confirmar, o la rama va por delante de `origin/<rama>` (o esa referencia no existe).
4. **Exclusiones**:
   - almacén `engram` o `none`;
   - sin aprobaciones ni momentos cumplidos (incluido `continuous` sin `docsPublication`);
   - compuerta `integration`;
   - en `continuous`, la ventana `apply`/`verify`. REQ-27.13 dice «No se publica tras `apply`», y el momento siguiente es el archivo.
5. **Multirepo.** En el repositorio de specs, la rama de publicación es la de integración in situ (`PublicationTarget.InPlace`), y el adelanto se mide contra `origin/<integración>`.
6. **Doctrina del orquestador** (`internal/assets/skills/_shared/sdd-orchestrator-sections.md`, párrafo de aprobación de compuerta y párrafo «When to publish», hoy en las líneas ~70 y ~80). Si el status proyecta publicación pendiente, se ejecuta su `invocation` exactamente una vez antes de lanzar la fase siguiente y al reanudar. Una negativa es informativa y no detiene la cadena. Se regeneran los goldens de `testdata/golden/` afectados.
7. **Delta de spec** en `sdd-docs-publication`: un requisito nuevo con sus escenarios.
8. **Pruebas localizadas** de los paquetes tocados: motor de ramas con git real y remoto desnudo local, `sddstatus`, `docpaths`, `assets` y goldens.

### Out of Scope

- Que `gate record` ejecute git. REQ-26.5 y REQ-27.11 se mantienen.
- El carril ODD, por decisión del usuario.
- Hooks que bloqueen el cierre de turno (el hook Stop de RDD se retira en la sección 01).
- El atraso (`behind`) respecto de `origin/<rama>`, `fetch` y cualquier operación remota. Eso es C-05-d.
- Cambiar la semántica de `PublicationTarget`, de `axiom sdd publish` o de la lista cerrada de REQ-27.5.
- Proyectar la publicación en la lista de cambios. Solo se calcula para el cambio seleccionado, para no multiplicar lecturas de git.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `sdd-docs-publication`: requisito nuevo «El status nativo proyecta la publicación pendiente». Recoge la condición (a)+(b), la ausencia sin red y la primera publicación como pendiente. También recoge la invocación idéntica a la de la compuerta, las exclusiones (almacén, `integration`, ventana `apply`/`verify` en `continuous`), el multirepo in situ y la obligación del orquestador: una ejecución antes de la fase siguiente y al reanudar, con la negativa informativa. REQ-27.12 (en `sdd-block-review-gates`) y REQ-27.13 no cambian de texto. El número lo fija `sdd-spec`: REQ-27.15 a REQ-27.17 ya se usan en otras capacidades.

## Approach

**Dónde vive el cálculo.** No puede vivir en `internal/cli` ni en `internal/sddpublish`. REQ-27.11 y `internal/sddpublish/isolation_test.go:204` fijan que solo `internal/cli/sdd_publish.go` importa `sddpublish`, y `sddstatus` ya depende del motor (`governance_handoff.go:97-115`). Se propone repartirlo así:

| Pieza | Ubicación | Motivo |
|-------|-----------|--------|
| Hechos de git (docs sucias, commits sin subir, existencia de la referencia remota) | `internal/branch/engine`: un método de lectura nuevo junto a `PublicationTarget` (por ejemplo, `publication_pending.go`) | El puerto `branch.Git` ya ofrece `DirtyPaths`, `ResolveCommit`, `AheadBehind`, `RevList` y `CommitFiles`, sin `push` ni `commit`. Reutiliza `gather` y el destino ya calculado. |
| Lista cerrada de rutas | `internal/docpaths`, paquete hoja y puro | Se extrae ahí la composición que hoy hace `composeSet` en `sddpublish/publish.go:318-368`. `sddpublish` delega en ella, así que `publish` y el status no pueden divergir. |
| Condición (a), exclusiones y proyección JSON | `internal/sddstatus` | La gobernanza (compuertas, kickoff, almacén, fase) ya vive ahí. Reutiliza `resolveGatePublication` para la invocación. |

No hay ciclos de importación:

- `sddstatus` importa `branch/engine`, y este importa `docpaths`, que es hoja;
- `sddpublish` importa `docpaths`;
- ni `engine` ni `docpaths` importan `sddstatus` ni `sddpublish`.

**Decisión pendiente D1 (producto, para el orquestador).** Hay que decidir qué proyecta el status cuando los commits sin subir incluyen código. En ese caso `publish` se niega con `code_commits_unpublished` (REQ-27.8), por ejemplo tras aprobar `role-apply` en `checkpointed`.

- **Opción A:** proyectar `pending` igualmente. Es más simple, pero el orquestador repetirá una negativa ya conocida en cada transición y en cada reanudación hasta que la persona suba el código.
- **Opción B (recomendada):** proyectar un estado distinto, informativo y sin ejecución, con la orden de push para la persona (`git push origin <rama>:<rama>`). Aun así, se proyecta `pending` mientras haya documentación sin confirmar, para que `publish` la confirme en local. Cuesta unas decenas de líneas (clasificar con `CommitFiles` y `docpaths.Set.Contains`) y evita el bucle.

Esta decisión solo afecta al punto 1 (clasificación de código) y a sus escenarios. El resto de la propuesta es independiente.

**Momentos `auto` cumplidos.** Se derivan del estado de las fases que ya proyecta el status (artefacto de `spec`, `design` o `tasks` completo; archivado). `sdd-design` confirma que el status resuelve el cambio archivado y la rama de archivado (`PublicationRequest{Archive: true}`). Si no los resuelve, el momento de archivo queda documentado como límite y no se proyecta.

**Invariantes.**

- Nunca hay red ni escrituras.
- Un fallo de lectura de git o una negativa de `PublicationTarget` nunca hace fallar el status. Se proyecta, como en la compuerta, el código de indisponibilidad, o nada.
- `nextRecommended` y las dependencias no cambian.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/branch/engine/` (fichero nuevo junto a `publication.go`) | New | Hechos de publicación de solo lectura sobre el destino de `PublicationTarget`. |
| `internal/branch/publication.go` | Modified | Tipo de resultado de los hechos de publicación. |
| `internal/docpaths/docpaths.go` | Modified | Composición de la lista desde las rutas declaradas (extraída de `sddpublish`). |
| `internal/sddpublish/publish.go` | Modified | `composeSet` delega en `docpaths`, sin cambio de comportamiento. |
| `internal/sddstatus/governance.go`, `governance_handoff.go`, `status_v2.go` | Modified | Condición, exclusiones y proyección `publication` en `governance`. |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modified | Regla de ejecución única antes de la fase siguiente y al reanudar. |
| `testdata/golden/*` | Modified | Goldens que incluyen esa sección. |
| `openspec/changes/publicacion-pendiente/specs/sdd-docs-publication/spec.md` | New | Delta con el requisito nuevo. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Bucle de negativas `code_commits_unpublished` tras `role-apply` o con commits de código | High (sin D1) | Opción B de D1 o, con la opción A, la regla de «una vez por transición o reanudación». |
| Publicar tras `apply` en `continuous`, en contra de REQ-27.13 | Med | Excluir la ventana `apply`/`verify` en `continuous` y añadir un escenario explícito. |
| Deriva entre la lista de rutas del status y la de `publish` | Med | Una sola composición en `docpaths`, usada por ambos, con prueba de paridad. |
| Solape con C-06-d (escribe en `sddpublish`) y con C-03-b, C-06-b/c/d (`sddstatus`) | Med | La edición en `sddpublish` se limita a delegar `composeSet`. Hay que avisar al orquestador del orden de fusión. |
| Una carpeta del cambio sin seguimiento aparece en `git status` como un único `dir/`, y la clasificación por ruta puede tomar como documentación lo que solo es estado local | Low | `sdd-design` define cómo tratar las entradas con «/» final. Se cubre con una prueba con git real. |
| Coste de lecturas de git en cada `status` | Low | Solo para el cambio seleccionado y solo si se cumple la condición (a). |
| Cambio de semántica de `PublicationTarget` con la promoción de C-02-a | Low | Ya lo fija `TestPublicationAndHandOffKeepTargetingTheIntegrationInBranchMode` (`internal/branch/engine/promotion_contracts_test.go:17-43`): publica `feat/x` hacia `develop`, nunca `main_branch`. El cálculo nuevo se apoya en ese destino sin tocarlo. |

## Rollback Plan

Revertir el commit o la PR del cambio. Es aditivo:

- el campo JSON nuevo es opcional y se omite cuando no aplica;
- la extracción a `docpaths` no cambia el comportamiento de `publish`;
- la doctrina vuelve a su texto anterior al regenerar los goldens.

No hay datos persistidos que migrar: `gates.yaml` y `kickoff.yaml` no cambian de formato.

## Dependencies

- C-05-b fusionado: REQ-27.13 ya nombra los cuatro momentos. Verificado en `openspec/specs/sdd-docs-publication/spec.md:300-356`.
- C-02-a: verificado que no cambia el destino de publicación (ver la tabla de riesgos).
- C-01-e: verificado que `internal/kickoff/rdd_boundary_test.go` ya no existe. No hay guarda de vocabulario RDD en `internal/kickoff` que respetar.
- Decisión D1 del orquestador o del usuario antes de cerrar los escenarios de código sin subir en `sdd-spec`.

## Success Criteria

- [ ] Con una compuerta (no `integration`) aprobada y documentos sin confirmar, `axiom sdd status --json` proyecta `publication: pending` con la invocación idéntica a `publication.invocation` de la compuerta.
- [ ] Tras un `axiom sdd publish` correcto, deja de proyectarse.
- [ ] Con la rama por delante de `origin/<rama>`, o sin esa referencia remota, se proyecta.
- [ ] Con `engram` o `none`, sin aprobaciones, en `continuous` sin `docsPublication` y durante `apply`/`verify` en `continuous`, no se proyecta.
- [ ] En multirepo, el adelanto se mide contra la integración in situ del repositorio de specs.
- [ ] El cálculo no registra `fetch`, `pull`, `ls-remote`, `commit` ni `push` en el observador del punto único de git, y `isolation_test.go` sigue en verde.
- [ ] La doctrina ordena una ejecución antes de la fase siguiente y al reanudar, y los goldens están regenerados.
- [ ] Pruebas localizadas en verde: `go test ./internal/branch/engine/... ./internal/sddstatus/... ./internal/docpaths/... ./internal/sddpublish/... ./internal/assets/...` (con `-run` donde proceda).
