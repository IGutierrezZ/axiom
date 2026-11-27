# Propuesta: Registrar decisiones de puerta desde el dashboard (C-06-f)

## Intención

Hoy el dashboard solo lee `gates.yaml` (`internal/dashboard/service.go:425`, `kickoff.LoadGates`) y el detalle del incremento no tiene ninguna noción de puerta. La persona revisa en la UI el HTML explicativo (C-06-b) y la batería de pruebas (C-06-c), pero para decidir tiene que volver a la terminal o al agente. Esa ruptura invita a saltarse la revisión.

El cambio permite decidir, desde el detalle del incremento, la parada que el status nativo proyecta en un cambio `checkpointed`, con las tres salidas de las paradas y con motivo obligatorio al iterar. La decisión de producto está confirmada (P3): la publicación (commit de los documentos del cambio y push a `origin`) también se dispara desde la UI, tras un aviso explícito de operación remota y solo con el clic de la persona.

Regla dura: la puerta se registra SIEMPRE con la misma lógica que `axiom sdd gate record`, y la publicación con la de `axiom sdd publish`. No se duplica ninguna regla.

## Alcance

### Dentro del alcance

- **Núcleo único de registro en `internal/cli`.** Extraer el cuerpo de `runSDDGateRecord` (`internal/cli/sdd_gate.go:55-231`) a una función reutilizable (nombre orientativo `RecordGate(ctx, GateRecordRequest) (GateRecordOutcome, error)`) que conserva: validación de argumentos y de `--commit-review`, resolución de la raíz del cambio, guarda de rama (`guardChangeWriter`, contexto `archive` para `integration`), `requireHandOffStops`, `requireCommitReviewStops`, digest del artefacto, preparación de la cesión, verificación de evidencia, `AppendGate`/`AppendGateChecked` bajo cerrojo, el evento de integración de C-07-e (una sola vez) y el cálculo de `publish`/`publishUnavailable`. Los avisos que hoy van a stderr pasan a un campo del resultado. El verbo CLI llama al núcleo y solo formatea; el contrato `gate-record/v1` y su salida no cambian.
- **Publicación en proceso reutilizable.** Una función `PublishChange` junto a `MarksPublisher` en `internal/cli/sdd_publish.go`, sobre el mismo `sddpublish.Publisher`, con `--branch` atado a la rama que la UI mostró. Sin `Only` ni `Replay`, sin fetch, nunca forzado.
- **Puerto del dashboard.** `dashboard.GatesGateway` con tres operaciones (`Stop`, `Record`, `Publish`), implementado en `internal/cli` e inyectado desde `cmd/axiom/main.go`, como `MarksPublisher` e `IntegrationsGateway`. `Stop` se apoya en `sddstatus.Resolve` para no recalcular `nextRecommended` ni la `GateQuestion`.
- **Rutas locales.**
  - `GET` de la parada actual del cambio: devuelve la `GateQuestion` proyectada (puerta, criterios, evidencia, opciones con sus etiquetas y efectos, `publication{remote, branch, prTarget, invocation}`) y el estado de publicación (`publication.state`), o «sin parada» cuando no la hay.
  - `POST /api/increments/gate` (registra) y `POST /api/increments/gate/publish` (publica), ambas bajo el preflight de rutas mutantes de las marcas (405, `Host`/`Origin` locales 403, `application/json` 415, 16 KiB, `DisallowUnknownFields`), con `confirm` obligatorio (422 antes de cualquier efecto), cerrojo (`busy`) y negativas tipadas (409).
  - Comprobación de «puerta aún pendiente» bajo cerrojo en todo registro desde la UI: la petición lleva la puerta y el digest vistos en pantalla; si la puerta ya se decidió o el digest cambió, se rechaza sin escribir.
- **Presentación en el detalle del incremento.**
  - (a) Una parada a la vez, la que proyecta el status (`nextRecommended == await-gate`), incluidas las `role-apply:<rol>` de cada rol en multirepo; la UI no inventa orden propio.
  - Las tres salidas de las paradas `spec`, `design`, `tasks` y `role-apply:<rol>`, con las etiquetas y efectos que proyecta la `GateQuestion` (fuente única, RF-GAT-14): Iterar (rechazo con motivo obligatorio); Aprobar, publicar y parar (`--hand-off` y publicación); Aprobar, publicar y continuar (aprobación y publicación). Cuando la `GateQuestion` no ofrece la salida 2 (almacén `engram|none`, RF-GAT-12), la UI tampoco.
  - (b) `integration` con sus dos salidas: aprobar con la evidencia de merge que exige la CLI (reutilizando su validación) o rechazar con motivo. No publica.
  - (c) Al aprobar `tasks` en `checkpointed`, una interacción propia con el modo de revisión por commit (`notify` por defecto, `stop`, `stop-before-commit`), registrado en la misma llamada de aprobación, como `--commit-review`. No se ofrece al iterar.
  - En las salidas 2 y 3, un único clic de confirmación en `confirmDialog` que avisa explícitamente de la operación remota (remoto y rama) encadena registrar y publicar.
  - Un fallo de publicación nunca deshace la aprobación: se muestra tipado, con su continuación o `pushCommand` (por ejemplo `code_commits_unpublished`), y con un botón de reintento mientras el status proyecte `publication.state: pending`.
  - En `continuous`, o sin `gateQuestion`, la UI no ofrece ninguna decisión.
- **Guardas de calidad existentes.** Registrar las rutas nuevas en el canario de secretos (`secrets_canary_test.go`), registrar el nuevo escritor en el inventario (`writer_inventory_test.go`), mantener el trinquete de rechazos (`refusal_resolution_ratchet_test.go`) y la CSP sin estilos ni scripts en línea.
- **Plan.** Actualizar la entrada C-06-f de `docs/plan/backlog-2026-11/06-ui-human-loop.md` con las decisiones confirmadas y su criterio de terminado.

### Fuera del alcance

- Cambiar la semántica de las puertas, su orden, sus digests, el contrato `gate-record/v1` o el texto de las tres salidas de `stop-exits.md`.
- Decidir puertas en cambios `continuous` o en el carril ODD.
- Escribir en `gates.yaml` desde el dashboard sin pasar por el núcleo (`AppendGate` directo).
- Lanzar desde la UI la revisión independiente (verify por rol, judgment, revisión de cobertura) o la verificación conjunta de `integration`; la UI solo muestra lo que el status ya proyecta.
- Los verbos del tracker que el orquestador ejecuta tras una aprobación (`axiom integrations summary push`, `tasks create`, `time propose|record`). Solo se conserva el evento de integración que ya dispara `gate record`.
- `fetch`, push de código, PR o merge desde la UI. Ejecutar `axiom autofetch run`.
- Editar la identidad del actor desde la UI.

## Capacidades

### Nuevas capacidades

- Ninguna.

### Capacidades modificadas

- `local-web-dashboard`: requisitos nuevos (`prov:WEB:<n>`) para la consulta de la parada actual, las rutas de registro y de publicación de puertas (preflight local, confirmación, cerrojo, comprobación de puerta pendiente, negativas tipadas), la presentación de las tres salidas, las dos de `integration` y el modo de revisión por commit, el estado «aprobada sin publicar» con reintento, la ausencia de decisiones en `continuous`, y la clasificación de las rutas en el canario de secretos.
- `sdd-block-review-gates`: un requisito nuevo (`prov:GAT:<n>`) que declara el dashboard como canal humano equivalente a `axiom sdd gate record`: comparte el mismo núcleo, produce registros indistinguibles en `gates.yaml`, emite el evento de integración una sola vez y aplica la misma guarda de rama. Ningún requisito existente cambia; la doctrina de los agentes sigue registrando solo con `axiom sdd gate record`.

## Enfoque

Enfoque 1 de la exploración: extraer el núcleo y exponerlo mediante un puerto, sin reimplementar nada en el dashboard.

1. **Refactor sin cambio de comportamiento.** `runSDDGateRecord` pasa a ser un adaptador fino sobre `RecordGate`. Los tests actuales del verbo deben pasar sin cambios de expectativas. Los rechazos que se muevan conservan su nombre de salida o su marca `// refusal:by-design`, y se actualiza la línea base del trinquete.
2. **Puerto y adaptador.** `GatesGateway` vive en `internal/dashboard` (dirección `dashboard -> cli` ya legal); la implementación en `internal/cli` se inyecta desde `cmd/axiom/main.go`, siguiendo `SetMarksPublisher`.
3. **Rutas y servicio.** Se reutilizan el patrón de `server_marks.go` (preflight) y de `service_marks.go` (`confirm`, `TryLock`, negativas 409).
4. **UI.** Un módulo de pantalla para la parada dentro del detalle, con `confirmDialog`/`promptDialog` y el patrón de botón con aviso de `detalle-pruebas.js`.
5. **Dos pasos no atómicos a la vista.** Registrar y publicar son dos llamadas. El estado intermedio se proyecta desde el status (`publication.state`) y se puede reintentar sin volver a registrar.

Se descartan ejecutar `app.RunArgs` en proceso (frágil, errores no tipados) y reimplementar con `kickoff.AppendGateChecked` (duplica digest, guarda, evidencia, cesión y evento).

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/cli/sdd_gate.go` | Modificado | `runSDDGateRecord` delega en el núcleo extraído |
| `internal/cli/` (fichero nuevo del núcleo, p. ej. `sdd_gate_core.go`) | Nuevo | `RecordGate`, petición y resultado tipados, avisos como datos |
| `internal/cli/sdd_gate_handoff.go`, `sdd_gate_commit_review.go` | Modificado | Ajustes para servir al núcleo sin cambiar reglas |
| `internal/cli/sdd_publish.go` | Modificado | `PublishChange` junto a `MarksPublisher` |
| `internal/cli/` (adaptador del puerto) | Nuevo | Implementación de `GatesGateway` (`Stop` sobre `sddstatus.Resolve`, `Record`, `Publish`) |
| `internal/dashboard/` (puerto, `server_gates.go`, `service_gates.go`) | Nuevo | Puerto, rutas GET/POST y servicio con cerrojo y comprobación de pendiente |
| `internal/dashboard/server.go` | Modificado | Registro de rutas |
| `internal/dashboard/assets/js/screens/detalle.js` y módulo nuevo de puertas | Modificado/Nuevo | Parada, salidas, diálogos, estado de publicación |
| `cmd/axiom/main.go` | Modificado | Inyección del puerto |
| `internal/dashboard/secrets_canary_test.go`, `writer_inventory_test.go`, `refusal_resolution_ratchet_test.go` (línea base) | Modificado | Registro de rutas, del escritor y de rechazos movidos |
| `openspec/changes/puertas-desde-ui/specs/` | Nuevo | Deltas de `local-web-dashboard` y `sdd-block-review-gates` |
| `docs/plan/backlog-2026-11/06-ui-human-loop.md` | Modificado | Entrada C-06-f con decisiones y criterio de terminado |

Paquetes `internal/` afectados: `internal/cli`, `internal/dashboard` (y lectura de `internal/kickoff`, `internal/sddstatus`, `internal/sddpublish` sin cambios previstos).

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| Regresión del verbo `gate record` al extraer 175 líneas con muchos tests | Media | Refactor primero y sin cambios de expectativas; tests del verbo intactos antes de añadir la UI |
| Doble evento de integración (spec, design, role-apply) | Baja | Un único punto de emisión en el núcleo; prueba que cuenta un solo `Dispatch` por aprobación |
| Carrera UI frente a CLI o agente (aprobación oculta, RF-GAT-09) | Media | Puerta y digest vistos en la petición, comprobados bajo el cerrojo del ledger en todo registro |
| Rama equivocada al registrar desde la UI | Baja | La guarda de rama vive en el núcleo; el inventario de escritores falla si falta el registro |
| Publicar es remoto (commit y push) | Media | Aviso explícito con remoto y rama, `confirm` obligatorio, rama atada a la mostrada, sin fetch y nunca forzado |
| Estado «aprobada sin publicar» tras un fallo | Media | Se muestra tipado con reintento a partir de `publication.state: pending`; la aprobación nunca se deshace |
| Desplazamiento de coordenadas en la línea base del trinquete | Alta | Actualizar la línea base en la misma unidad de trabajo del refactor |
| Aprobar desde la UI no ejecuta los verbos del tracker que el orquestador lanza tras la puerta | Media | Fuera de alcance declarado; decisión de producto devuelta al orquestador |
| Orden de merge con C-08-c y C-01-e, que tocan primitivas relacionadas | Media | Confirmar el orden antes de `apply`; rebase sobre `origin/main` si se adelantan |

## Plan de reversión

El cambio entra en una sola PR con un commit por tarea. Para revertir: `git revert` de la fusión de la PR. Si solo falla la UI, basta con revertir los commits de rutas, servicio y assets: el núcleo extraído es un refactor sin cambio de comportamiento y la CLI sigue funcionando igual. Los registros que la UI haya escrito en `gates.yaml` son idénticos a los de la CLI, así que no hay datos que migrar ni limpiar; un commit de publicación ya subido se trata como cualquier otro commit de documentos del cambio.

## Dependencias

- C-06-a (detalle del incremento) y las paradas de la sección 02 (`--hand-off`, `--commit-review`), ya en `main`.
- C-07-e (eventos de integración al aprobar), ya en `main`: se reutiliza, no se modifica.
- C-05-c (`publication.state` en el status), para el reintento de la publicación.
- Coordinación con C-08-c y C-01-e por el orden de merge.

## Criterios de éxito

- [ ] Desde el detalle de un cambio `checkpointed`, la persona decide `spec`, `design`, `tasks` y cada `role-apply:<rol>` con las tres salidas, y `integration` con sus dos, sin usar la terminal.
- [ ] Un registro hecho desde la UI es indistinguible en `gates.yaml` del hecho con `axiom sdd gate record` con los mismos datos, y emite el evento de integración una sola vez.
- [ ] Iterar sin motivo se rechaza sin escribir; aprobar una puerta ya decidida o con el digest cambiado se rechaza con 409 sin escribir.
- [ ] Las salidas 2 y 3 publican solo tras confirmar el aviso de operación remota; un fallo de publicación deja la aprobación registrada y ofrece reintento.
- [ ] En `continuous`, o sin parada proyectada, la UI no ofrece ninguna decisión.
- [ ] Los tests existentes de `gate record` pasan sin cambios de expectativas; el canario de secretos, el inventario de escritores, el trinquete de rechazos y la prueba de CSP siguen en verde.
