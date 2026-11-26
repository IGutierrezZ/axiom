# Design: Marcas de ejecución por entorno de `pruebas.md` publicadas a la integración del repo de specs (C-06-d)

> **Cambio:** `marcas-pruebas`. **Idioma:** castellano de España; identificadores, claves, códigos y tokens en inglés.
> **Entradas:** `proposal.md`, `exploration.md`, deltas de `specs/` y `coverage-review.spec.md` (ronda 1).

## Enfoque técnico

Enfoque A de la propuesta, concretado en cinco piezas con una única fuente de cálculo:

1. **Núcleo puro de marcas** en un subpaquete nuevo, `internal/acceptancetests/marks`: formato de `pruebas-estado.yaml`, codificación segura, validación de una marca contra la batería, resumen por entorno, entradas pendientes y fusión para la reproducción. No lee ficheros ni ejecuta git.
2. **Lectura compartida en `internal/sddstatus`** (`AcceptanceMarksFor`): lee el fichero, calcula `plan_digest` con `kickoff.ArtifactDigest` y obtiene la versión publicada del fichero a través del motor de ramas (solo lecturas). La usan `axiom sdd status` (proyección `acceptanceTests.marks`), el dashboard (pestaña «Pruebas») y, vía status, la puerta `integration` y el `archive-report`: un solo cálculo para las tres presentaciones (RF-KOF-22 modificado).
3. **Modo acotado en `internal/sddpublish`**: `Request.Only` (lista exacta, solo `pruebas-estado.yaml`) y `Request.Replay` (reintento único por reproducción ante `push_non_fast_forward`, solo junto con `Only`). `axiom sdd publish` no expone ninguno de los dos (RF-PUB-01, RF-PUB-05 y RF-PUB-07 modificados; prov:PUB:1 y prov:PUB:2).
4. **Publicador de marcas en `internal/cli/sdd_publish.go`**, inyectado en el dashboard desde `cmd/axiom/main.go` a través de una interfaz propia del dashboard. El dashboard no importa `sddpublish` ni ejecuta git (prov:WEB:3).
5. **Dashboard**: dos rutas POST (`/api/increments/marks` y `/api/increments/marks/publish`), escritura local de solo añadir con cerrojo por cambio, estado «no disponible» tipado y la pestaña «Pruebas» ampliada con la matriz de marcas (prov:WEB:1 a prov:WEB:5, RF-WEB-17 modificado).

La doctrina (`sdd-archive` y la sección del orquestador sobre `integration`) solo copia el texto `summary` que calcula el status; no recalcula nada (prov:GAT:1, prov:ARC:1).

## Decisiones de arquitectura

### Decisión D1: subpaquete puro `internal/acceptancetests/marks`

**Elección**: paquete nuevo `marks` bajo `internal/acceptancetests/`, que importa solo `internal/acceptancetests` (tipos `Battery`/`Case`), `gopkg.in/yaml.v3` y la biblioteca estándar, con su propia prueba de importaciones (sin `os/exec`, sin `os` para E/S, sin otros paquetes internos).
**Alternativas consideradas**: (a) añadir los ficheros a `internal/acceptancetests`; (b) modelo dentro de `internal/sddstatus`; (c) YAML escrito a mano.
**Justificación**: `internal/acceptancetests/imports_test.go:30` (`TestPackageImports`) prohíbe paquetes externos en el núcleo de la batería y su doc (`doc.go:6-8`) promete que solo depende de `reqid` e `isoevidence`; ampliar esa regla para meter `yaml.v3` debilita un contrato existente. Un subpaquete mantiene la localidad y es reutilizable desde `sddstatus`, `dashboard` y `cli` sin ciclos. (b) impediría que `cli` (fusión de la reproducción) lo use sin arrastrar `sddstatus`. (c) es frágil para notas con `:`, `#`, comillas o Unicode (hueco 3).

### Decisión D2: formato de `pruebas-estado.yaml` como secuencia YAML de nivel superior, de solo añadir por concatenación

**Elección**: el fichero es una secuencia de mapas, sin cabecera:

```yaml
- case: P-01
  environment: pre
  result: ok
  by: Ana Pérez <ana@example.org>
  at: "2026-11-26T19:18:33.072Z"
  commit: ""
  plan_digest: 2fe1cfd9...
  note: "Probado con datos reales: OK"
```

Añadir una marca es concatenar los bytes existentes (con `\n` final garantizado) y la codificación `yaml.v3` de `[]Entry{e}`. El contenido previo es siempre prefijo exacto del nuevo.
**Alternativas consideradas**: mapa con `schema:` y `entries:`; reescritura completa del fichero tras parsear.
**Justificación**: la concatenación hace verificable byte a byte la regla de solo añadir (prov:KOF:1), es estable ante fusiones de texto de git y hace trivial la fusión de la reproducción (remoto + entradas locales que faltan). Una cabecera no aporta nada que el nombre del fichero no fije y obliga a que `entries:` sea siempre la última clave. Reescribir tras parsear rompería el «byte a byte» y podría normalizar entradas ajenas.

### Decisión D3: identidad de entrada y definición de «pendiente de publicar» (hueco 1)

**Elección**:
- Dos entradas son **la misma** cuando sus ocho campos son iguales (`case`, `environment`, `result`, `by`, `at`, `commit`, `plan_digest`, `note`). `at` lleva milisegundos en UTC (`2006-01-02T15:04:05.000Z`) y el servicio garantiza que dos marcas escritas por el mismo proceso nunca comparten `at` (si coincide con la última escrita, suma 1 ms).
- **Versión publicada** = el contenido del fichero en `refs/remotes/origin/<rama publicada>` (en multirepo, la integración del repo de specs), leído sin red. Sin esa referencia o sin el fichero en ella, la versión publicada es vacía.
- **Pendientes** = diferencia de multiconjuntos `local − publicada`, en el orden del fichero local. Las entradas confirmadas pero no subidas siguen siendo pendientes; las que trajo un `git pull` de otra persona no lo son.
- La **fusión de la reproducción** (`marks.Merge(local, remote)`) devuelve `remote` + las entradas de `local − remote`, codificadas en orden local. Nunca duplica: lo que ya está en el remoto, aunque lo subiera un push que el cliente creyó fallido, no se vuelve a añadir.
**Alternativas consideradas**: comparar contra `HEAD`; registro local de publicadas en un fichero con punto; campo `id` nuevo por entrada.
**Justificación**: `HEAD` marcaría como publicadas entradas confirmadas que nunca subieron (fallo de push). Un registro local diverge entre máquinas y tras `pull`. Un `id` añade un campo que la spec no define. La referencia de seguimiento la actualiza el propio push con éxito y el `fetch` de la reproducción, así que «0 pendientes» tras publicar es observable sin red.

### Decisión D4: semántica del resumen por entorno (hueco 4)

**Elección** (`marks.Summarize`, única implementación):
- **Entornos**: los de `acceptance_tests.environments`, en su orden declarado.
- **Casos**: filas de `pruebas.md` (`acceptancetests.Parse`, `internal/acceptancetests/parse.go:92`) con ID válido `P-NN`; un ID repetido cuenta una vez y sus entornos aplicables son la unión de sus filas. **Entornos aplicables del caso** = columna `Entornos` (`Case.Environments`, `internal/acceptancetests/model.go:64`, separada por `;` en `parse.go:126`) ∩ entornos declarados.
- **Total del entorno E** = número de casos aplicables a E. Con total 0 el entorno se muestra como `E: sin casos`.
- **Marca vigente del par (caso, E)**: entre las entradas con ese par y `plan_digest` igual al digest actual, la de mayor `at`; a igualdad de `at`, la última del fichero.
- **Categorías disjuntas por par aplicable**: `ok`, `ko`, `blocked`, `na` (marca vigente con ese resultado); `stale` (sin marca vigente pero con alguna entrada caducada del par); `unmarked` (ninguna entrada). Cumple `ok + ko + blocked + na + stale + unmarked = total`. `n/a` permanece en el total y no cuenta como `ok`.
- **Huérfanas**: entradas cuyo caso no existe ya en `pruebas.md`, o cuyo entorno no está declarado o no es aplicable al caso. No cuentan en ninguna categoría; se proyectan como `orphans`.
- **Entradas inválidas** (YAML válido, campos fuera de forma: resultado fuera de dominio, `at` no parseable, digest no hexadecimal de 64, `by` vacío): se ignoran y se cuentan en `invalid`. Un YAML ilegible o que no es una secuencia invalida el fichero entero (`error`, sin recuentos).
- **Texto canónico** (`Summary.Text()`): por entorno, `E: <ok>/<total> ok` y, separados por `, `, solo las categorías no nulas: `<n> ko`, `<n> bloqueado(s)`, `<n> n/a`, `<n> sin marcar`, `<n> caducada(s)`; entornos separados por `; `. Ejemplo del escenario de RF-KOF-22: `pre: 5/5 ok; local: 1/2 ok, 1 caducada`. Las tres presentaciones copian este texto literal.
**Alternativas consideradas**: excluir `n/a` del denominador; «última» por orden del fichero; contar todas las entradas caducadas.
**Justificación**: excluir `n/a` del total haría que «5/5 ok» oculte un caso no ejecutado; mostrarlo aparte es honesto («nunca como limpio»). Tras una reproducción el orden del fichero mezcla personas, así que `at` refleja mejor cuál es posterior; el desempate por orden cubre `at` iguales. Contar todas las caducadas crecería con el historial; contar pares sin marca vigente es lo que el escenario `local: 1 de 2 ok y 1 caducada` exige.

### Decisión D5: validación y serialización de la marca (huecos 2 y 3)

**Elección**:
- `case`: debe existir en la batería actual → si no, `marks_case_unknown`.
- `environment`: debe estar declarado y ser aplicable al caso → si no, `marks_environment_invalid`.
- `result`: `ok|ko|blocked|n/a` exactos → si no, `marks_result_invalid`.
- `note`: se recorta; UTF-8 válido; sin caracteres de control (U+0000–U+001F, U+007F; incluye CR, LF y TAB) → `marks_note_invalid`; como máximo **500 runas** → `marks_note_too_long`.
- `commit`: opcional, se recorta; vacío por defecto; como máximo **100 runas** y solo `[A-Za-z0-9._/+:#@-]` (SHA, etiqueta, `build#123`, `v1.2.3`) → `marks_commit_invalid`. Nunca se rellena desde el repo de código.
- `by`: `Identity.Actor()` de `internal/gitidentity` (`gitidentity.go:98`, saneado: sin controles ni `<`/`>` en las partes). **Sin identidad completa, la marca se rechaza** con `identity_missing` (409), nombrando las claves que faltan y las órdenes `git config` que las fijan; no se escribe nada.
- Serialización: solo con `yaml.v3` sobre el `struct Entry` (orden de campos fijo). Prueba de ida y vuelta con notas hostiles (`: `, `#`, comillas, `- `, `yes`, `null`, `<script>`, Unicode, espacios iniciales).
**Alternativas consideradas**: `by: cli` como `gate record` cuando falta identidad (`internal/cli/sdd_actor_identity.go:41`); notas multilínea.
**Justificación**: una marca sin autor real es inútil como evidencia y además no podría publicarse (la publicación exige identidad, prov:PUB:1); rechazar pronto evita escribir algo que luego bloquea. La UI usa un `input` de una línea; prohibir controles elimina ambigüedades de YAML y de presentación.

### Decisión D6: copia del repo donde se escribe y lee (hueco 7)

**Elección**: tanto la escritura de marcas como su lectura en el dashboard usan `filepath.Join(location.PlanningHome, "changes", <x>)` de `workspace.Locate(<raíz del dashboard>)`, es decir, el repo de specs **in situ** del multirepo. Se ignora siempre el `worktreePath` registrado que `locateIncrement` prioriza (`internal/dashboard/service_listing.go:41-50`). El status lee `status.ChangeRoot`, que en multirepo es la carpeta del cambio del mismo planning home. La publicación exige `PublicationTarget.InPlace` (`internal/branch/publication.go:32`).
**Alternativas consideradas**: escribir donde `locateIncrement` resuelve el incremento.
**Justificación**: un worktree registrado obsoleto haría que la marca se escriba en una copia que la publicación (in situ) no ve. Una prueba fija que el resumen del DTO del detalle y el de `axiom sdd status` coinciden sobre el mismo fixture.

### Decisión D7: cerrojo compartido entre marca y publicación (hueco 6)

**Elección**: `Service.marksLocks` (`sync.Map` de la raíz absoluta del cambio a `*marksLock{mu sync.Mutex; publishing atomic.Bool}`):
- **Marca**: si `publishing` es verdadero → `marks_busy` (409) inmediato; si no, `mu.Lock()` (bloqueante; la sección crítica es leer, validar y reemplazar un fichero pequeño). Dos marcas simultáneas se serializan y ambas quedan íntegras.
- **Publicación**: `mu.TryLock()`; si falla (otra publicación, doble clic, otra pestaña o una marca en curso) → `marks_busy` (409) con continuación «espera a que termine y repite». Con el cerrojo, `publishing = true` hasta terminar.
- Alcance: un proceso. Dos dashboards a la vez sobre el mismo repo no están protegidos (`dashboardAlreadyRunning` evita el caso habitual); queda como riesgo aceptado.
**Alternativas consideradas**: cerrojo de fichero con `fileprim.AcquireLock`; serializar la publicación en cola.
**Justificación**: una publicación puede durar el tiempo máximo del push; bloquear una petición HTTP de marca durante ese tiempo es peor que un rechazo tipado inmediato. El cerrojo de fichero añadiría un fichero de estado y limpieza sin un caso real que lo pida.

### Decisión D8: contrato del publicador (interfaz del dashboard, implementación en `cli`)

**Elección**: el dashboard ya importa `internal/cli` (`internal/dashboard/service.go`, `service_adapters.go`, `service_agents.go`), así que `cli` no puede importar `dashboard`. La interfaz vive en el dashboard y sus tipos de petición y resultado en `cli`:

```go
// internal/dashboard/marks_port.go
type MarksPublisher interface {
	MarksIdentity(ctx context.Context, dir string) (actor string, missing []string)
	PublishMarks(ctx context.Context, req cli.MarksPublishRequest) cli.MarksPublishResult
}
func (s *Service) SetMarksPublisher(p MarksPublisher)
```

`cli.MarksPublisher` (en `internal/cli/sdd_publish.go`) la satisface; `cmd/axiom/main.go:1207` la inyecta con `svc.SetMarksPublisher(cli.NewMarksPublisher())` tras `NewServiceWithHub`. Sin inyección (pruebas, `NewService` de otros flujos) las marcas responden `marks_publisher_missing`.
**Alternativas consideradas**: fichero nuevo `internal/cli/sdd_publish_marks.go`; subproceso `axiom sdd publish --only` (enfoque B).
**Justificación**: dejarlo en `sdd_publish.go` mantiene intacta `TestIsolationOnlyTheCLIVerbImportsSddpublish` (`internal/sddpublish/isolation_test.go:204`), cuyo único importador permitido es ese fichero; no hay que tocar el trinquete. B se descartó en la propuesta (binario en ejecución, JSON, peor prueba).

### Decisión D9: modo acotado de `sddpublish` (`Only` y `Replay`)

**Elección**: `Request` gana dos campos no accesibles desde el verbo:

```go
type Request struct {
	CWD, Change string
	Archive     bool
	Branch      string
	// Only: nombres relativos a la carpeta del cambio; solo docpaths.ExactAllowed.
	Only []string
	// Replay: fusión para el reintento por reproducción; exige Only.
	Replay func(local, remote []byte) ([]byte, error)
}
```

Comportamiento de `Publisher.Run` (`internal/sddpublish/publish.go:46`) con `Only`:
1. **Antes de cualquier git**: `Only` no vacío, sin `Archive`, cada nombre aceptado por `docpaths.ExactAllowed(name)` (solo `pruebas-estado.yaml`); `Replay` sin `Only` también se rechaza. Código `only_path_rejected`.
2. Pasos 1 a 5 sin cambios; tras el destino, `!target.InPlace` → `only_requires_in_place`.
3. Sucios: `DirtyPaths` sobre las rutas de `Only`; los demás documentos sucios de la lista cerrada (sin estado local) se devuelven en `Result.UncommittedDocs` como aviso, sin tocarlos.
4. **Antes del commit**: el rango sin subir solo puede contener commits que toquen rutas de `Only`; si no, `other_commits_unpublished` sin crear commit.
5. Commit de las rutas de `Only` con `Git.Commit` (`git add --all -- <ruta>` y `git commit --only -- <ruta>`), que deja el resto del índice y del árbol intactos.
6. Push. Ante `push_non_fast_forward` y `Replay != nil`, **un único** reintento (D10). Cualquier otro rechazo no reintenta.
`Result` gana `UncommittedDocs []string` y `Replayed bool` (`omitempty`): el JSON de `axiom sdd publish` no cambia.
**Alternativas consideradas**: un booleano `MarksOnly` con la ruta fijada dentro de `sddpublish`; validar `Only` contra la lista cerrada completa.
**Justificación**: la spec pide una lista exacta validada contra la lista cerrada y limitada a `pruebas-estado.yaml`; validar el nombre de forma pura cumple «rechazar antes de ejecutar git». Comprobar el rango antes del commit evita crear un commit que no se podría subir.

### Decisión D10: reintento por reproducción ordenado para no perder entradas (hueco 5)

**Elección**: ante el primer `push_non_fast_forward` en modo `Only`+`Replay`:

| Paso | Operación (puerto `Git`) | Si falla |
|---|---|---|
| R1 | `Fetch(ctx, dir, "origin", rama)` = `git fetch --no-tags origin <rama>` con el tiempo máximo del push | `remote_unavailable` (sin red o credenciales) o `replay_failed`; nada local cambia; el commit local se conserva |
| R2 | `FileAt(ctx, dir, "refs/remotes/origin/<rama>", ruta)` (`git cat-file blob <rev>:<ruta>`) y lectura del fichero local | `replay_failed`; nada cambia |
| R3 | `merged, err := Replay(local, remote)` | YAML remoto o local ilegible → `replay_failed`; nada cambia |
| R4 | `ResetKeep(ctx, dir, "refs/remotes/origin/<rama>")` = `git reset --keep <ref>` | aborta sin cambios si otro documento sucio choca con el remoto → `replay_failed` nombrando el conflicto; el commit local se conserva |
| R5 | escritura atómica de `merged` (temporal con punto en la raíz del cambio + `fileprim.ReplaceAtomic`) | `replay_failed` con continuación `git -C <raíz> show ORIG_HEAD:<ruta>` para recuperar |
| R6 | si `merged` no difiere del remoto → `up_to_date` (todo estaba ya publicado); si no, `Commit` + `Push` | segundo rechazo → su código tipado (`push_non_fast_forward`, `push_hook_rejected`, …) sin tercer intento; el commit nuevo se conserva (RF-PUB-05) |

- Las precondiciones del paso 4 de D9 garantizan que `reset --keep` solo descarta commits de marcas, cuyo contenido está en `local` (el fichero estaba limpio tras el commit).
- Ventana de caída entre R4 y R5: las entradas siguen en `ORIG_HEAD` y en el reflog; la continuación lo nombra. No se añade fichero de copia de seguridad.
- Tras agotar el reintento, la siguiente pulsación encuentra el fichero limpio y un commit de marcas sin subir, y vuelve a intentar con su propio reintento único.
- Sin `rebase`, sin `+` ni `--force`, sin `--no-verify`. `axiom sdd publish` pasa `Replay == nil` y la prueba existente «ninguna publicación hace `fetch`» (`internal/sddpublish/publish_test.go:496`) sigue en verde.
**Alternativas consideradas**: `git rebase`; `reset --soft`/`--mixed` y reescritura; fichero de copia previo al reset.
**Justificación**: `rebase` produce conflictos de texto en un YAML que dos personas amplían. `--soft`/`--mixed` dejan el índice o el árbol con reversiones aparentes de lo que trajo el remoto. `reset --keep` es atómico y se niega a pisar cambios locales. Ordenar las lecturas y la fusión antes del reset hace que todo fallo previo deje el estado intacto.

### Decisión D11: lectura de la versión publicada por el motor de ramas

**Elección**: método nuevo de solo lectura en el motor:

```go
// internal/branch/engine/published_file.go
func (e *Engine) PublishedChangeFile(ctx context.Context, req branch.PublicationRequest, name string) (branch.PublishedChangeFile, *branch.Refusal)

// internal/branch/publication.go
type PublishedChangeFile struct {
	Target          PublicationTarget
	RemoteRef       string // refs/remotes/origin/<Target.Branch>
	RemoteRefExists bool
	Exists          bool   // el fichero existe en RemoteRef
	Content         []byte
}
```

Reutiliza `resolvePublication` y el puerto `FileAt` ya existente (`internal/branch/engine/engine.go:52`, `internal/branch/gitcli/ops.go:258`). Sin red, sin escrituras.
**Alternativas consideradas**: leer la versión publicada en el publicador de `cli`; ejecutar git desde `sddstatus`.
**Justificación**: `sddstatus` ya obtiene hechos de git a través del motor (`publicationEngine()`, `internal/sddstatus/publication.go:57`); así el status, el dashboard y el archivo comparten la misma lectura sin que el dashboard ejecute git.

### Decisión D12: proyección en `sddstatus` y en el contrato

**Elección**: `AcceptanceTestsProjection` (`internal/sddstatus/acceptance_tests.go:64`) gana `Marks *AcceptanceMarksProjection \`json:"marks,omitempty"\``. `decorateAcceptanceTests` (`:187`) la rellena solo si la topología es `multirepo`, la función está activa, existe `pruebas.md` y existe `pruebas-estado.yaml`. No toca `NextRecommended`, `Dependencies`, `BlockedReasons`, `Notes`, `ISO`, `Governance.Gates` ni digests; `pruebas-estado.yaml` no entra en `TasksGateArtifactPaths` (`:26`).

```json
"marks": {
  "path": "<changeRoot>/pruebas-estado.yaml",
  "summary": "pre: 5/5 ok; local: 1/2 ok, 1 caducada",
  "environments": [{"environment":"pre","total":5,"ok":5,"ko":0,"blocked":0,"na":0,"unmarked":0,"stale":0}],
  "orphans": 0, "invalid": 0,
  "pending": 2,
  "pendingUnavailable": "wrong_branch",
  "error": "..."
}
```

`pending` solo aparece si la lectura publicada fue posible; si no, `pendingUnavailable` con el código del motor. `error` (fichero ilegible) sustituye a los recuentos y es informativo. `_shared/sdd-status-contract.md` documenta la clave.
**Alternativas consideradas**: proyectar las marcas también con la función inactiva.
**Justificación**: sin la función activa no hay dominio de entornos; omitir la clave es lo que la spec permite y no cambia el JSON de proyectos existentes.

### Decisión D13: rutas HTTP, cuerpos y códigos

**Elección**: dos rutas registradas en `server.go`, siguiendo el patrón de `/api/increments/verify` (ruta más específica que `/api/increments/`):

| Ruta | Cuerpo | Éxito | Rechazos |
|---|---|---|---|
| `POST /api/increments/marks` | `{"change","case","environment","result","commit"?,"note"?}` | 201 `{"entry":{…},"marks":{…}}` | ver tabla de códigos |
| `POST /api/increments/marks/publish` | `{"change","confirm":true,"branch"}` | 200 `{"status":"published"\|"up_to_date"\|"nothing_to_publish","remote","branch","commit"?,"pushed"?,"replayed","uncommittedDocs":[…],"marks":{…}}` | ver tabla de códigos |

Orden de comprobación en ambas: método POST (405) → `isLocalRequest` (`internal/dashboard/server.go:966`; 403, código `request_not_local`) → `Content-Type: application/json` (415, `marks_content_type`) → `http.MaxBytesReader` de 16 KiB y `DisallowUnknownFields` (400, `marks_body_invalid`) → en publicar, `confirm == true` (422, `marks_confirm_missing`, **antes de cualquier lectura de git**) → disponibilidad (409, D14) → cerrojo (D7) → validación (D5) → acción.

| Código | HTTP | Ruta | Continuación (resumen) |
|---|---|---|---|
| `request_not_local` | 403 | ambas | abre el dashboard en 127.0.0.1 o localhost |
| `marks_content_type` / `marks_body_invalid` | 415 / 400 | ambas | envía JSON con los campos documentados |
| `marks_confirm_missing` | 422 | publicar | confirma la publicación en el diálogo |
| `marks_case_unknown` / `marks_environment_invalid` / `marks_result_invalid` | 422 | marca | elige un caso, entorno o resultado de la tabla |
| `marks_note_too_long` / `marks_note_invalid` / `marks_commit_invalid` | 422 | marca | acorta o corrige el campo (límites en el DTO) |
| `identity_missing` | 409 | marca | `git config user.name …` y `git config user.email …` en el repo de specs |
| `marks_file_invalid` | 409 | ambas | corrige `pruebas-estado.yaml` o recupéralo con `git -C <repo> checkout -- <ruta>` |
| `marks_battery_unreadable` | 409 | marca | comprueba que `pruebas.md` se puede leer |
| `marks_busy` | 409 | ambas | espera a que termine y repite |
| D14 (`marks_topology_unsupported`, …) | 409 `{"status":"unavailable",…}` | ambas | la de cada código |
| códigos de `sddpublish` (`push_non_fast_forward`, `wrong_branch`, `identity_missing`, `remote_missing`, `replay_failed`, `other_commits_unpublished`, …) | 409 `{"status":"refused","code","label","error","continuation","details"?}` | publicar | la de `sddpublish` |

Todo cuerpo de rechazo lleva `code`, `error` (mensaje en castellano) y `continuation`. Sin entradas pendientes (D3), publicar devuelve `nothing_to_publish` sin llamar al publicador. `branch` del cuerpo es la rama que la UI mostró; se pasa como `Request.Branch`, de modo que un cambio de rama entre la vista y el clic da `wrong_branch`. En `secrets_canary_test.go:42` (`excludedRoutes`) se añaden ambas con «mutación (POST): …». La lectura sigue en `GET /api/increments/<n>` sin ruta nueva.
**Alternativas consideradas**: ruta GET de previsualización; `window.confirm`.
**Justificación**: RF-WEB-17 modificado prohíbe rutas GET nuevas; el remoto, la rama y las pendientes viajan en el DTO del detalle. Exigir JSON añade una barrera frente a envíos de formulario entre sitios (no hay token CSRF; riesgo heredado).

### Decisión D14: disponibilidad «no disponible» tipada

**Elección**: `marksAvailability` en `internal/dashboard/service_marks.go`, evaluada en este orden; la primera que falla da el código:

| Código | Condición |
|---|---|
| `marks_publisher_missing` | no se inyectó `MarksPublisher` |
| `marks_topology_unsupported` | `GoverningTopology()` distinta de `multirepo` (`internal/workspace/locate.go:64`) |
| `marks_not_sdd` | el nombre corresponde a una feature ODD, no a `openspec/changes/<x>` |
| `marks_change_archived` | el cambio solo existe archivado |
| `marks_not_active` | `acceptance_tests` no activo o con error de declaración |
| `marks_battery_missing` | no existe `pruebas.md` |

Un nombre que no existe en ningún sitio sigue siendo 404 (`change_not_found`): no es un caso de «no disponible». La topología va antes que la activación porque en monorepo la función nunca se activa (`internal/workspace/acceptance_tests.go:167`) y el motivo correcto es la topología.

### Decisión D15: pantalla en la pestaña «Pruebas» existente

**Elección**: se amplía `internal/dashboard/assets/js/screens/detalle-pruebas.js` (sin pestaña nueva; las pruebas de pines de pestañas no cambian):
- Con `marks.available == false`: la tabla de casos de siempre, de solo lectura, y un `stateEmpty` con el motivo del código.
- Con `marks.available == true`: matriz casos × entornos. Celda aplicable: resultado vigente (badge), nota y `commit` como texto escapado (`escapeHtml`), indicador «caducada: <resultado>» si solo hay marca caducada, y un formulario (selector `ok|ko|blocked|n/a`, `commit` y `note` con `maxlength` del DTO, botón «Marcar»). Celda no aplicable: «—» sin control.
- Bloque «Publicar marcas», siempre visible encima del botón: `Remoto: origin · Rama: <rama> · <n> entradas pendientes` y la lista de pendientes (caso, entorno, resultado, autor, fecha). El botón abre `confirmDialog` (`core/components.js:157`) con los mismos datos y solo al aceptar envía `{confirm:true, branch}`. Deshabilitado con 0 pendientes o con `pendingUnavailable`.
- Respuesta: `toast` con commit, rama y remoto o con `label`, `error` y `continuation` del rechazo; avisos de `uncommittedDocs`. Tras cada acción se repinta con el `marks` devuelto.
- DTO nuevo `IncrementAcceptanceTestsDTO.Marks *AcceptanceMarksDTO` (`types.go:172`) con `available`, `unavailable{code,error,continuation}`, `environments`, `cells[]`, `summary`, `summaryText`, `pending[]`, `pendingUnavailable`, `remote`, `branch`, `noteMax`, `commitMax`, `error`.

## Flujo de datos

```
  Navegador (pestaña «Pruebas»)
     │ GET /api/increments/<x>                    POST /marks            POST /marks/publish {confirm,branch}
     ▼                                               │                         │
  dashboard.Service ──► sddstatus.AcceptanceTestsFor │                         │
     │                  sddstatus.AcceptanceMarksFor ◄────────┐                 │
     │                     │  marks.Parse/Summarize/Pending   │                 │
     │                     └─► engine.PublishedChangeFile ──► git cat-file (solo lectura)
     │                                               │                         │
     │   validar (marks.ValidateInput) + identidad ◄─┤                         │
     │   (MarksPublisher.MarksIdentity → gitidentity)│                         │
     │   marks.Append + fileprim.ReplaceAtomic ◄─────┘                         │
     │                                                                         ▼
     │                                           MarksPublisher.PublishMarks (internal/cli/sdd_publish.go)
     │                                                       │ sddpublish.Publisher.Run{Only, Replay: marks.Merge}
     │                                                       ▼
     │                                           gitport: add/commit --only → push
     │                                             └─ non-ff: fetch → cat-file → Merge → reset --keep → write → commit → push
     ▼
  axiom sdd status ── acceptanceTests.marks.summary ──► presentación de `integration` / `archive-report`
```

## Cambios por fichero

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/acceptancetests/marks/doc.go`, `model.go`, `codec.go`, `summary.go`, `validate.go` | Crear | Núcleo puro: `Entry`, `Parse`, `Append`, `Merge`, `Pending`, `ValidateInput`, `Summarize`, `Summary.Text`, códigos de validación, límites |
| `internal/acceptancetests/marks/*_test.go` | Crear | Pruebas puras e `imports_test.go` propio |
| `internal/docpaths/docpaths.go` | Modificar | `ExactAllowed(name string) bool` (solo `pruebas-estado.yaml`) |
| `internal/branch/publication.go` | Modificar | Tipo `PublishedChangeFile` |
| `internal/branch/engine/published_file.go` | Crear | `Engine.PublishedChangeFile` |
| `internal/sddstatus/acceptance_marks.go` | Crear | `AcceptanceMarksFor`, `AcceptanceMarksResult`, `AcceptanceMarksProjection` |
| `internal/sddstatus/acceptance_tests.go` | Modificar | Campo `Marks` y su decoración |
| `internal/sddpublish/result.go` | Modificar | `Request.Only`, `Request.Replay`, `Result.UncommittedDocs`, `Result.Replayed`; códigos `only_path_rejected`, `only_requires_in_place`, `other_commits_unpublished`, `replay_failed` con etiqueta |
| `internal/sddpublish/ports.go`, `gitport.go` | Modificar | `Fetch`, `FileAt`, `ResetKeep` |
| `internal/sddpublish/publish.go` | Modificar | Ramas `Only` y reintento (D9, D10) |
| `internal/cli/sdd_publish.go` | Modificar | `MarksPublisher`, `NewMarksPublisher`, `MarksPublishRequest`, `MarksPublishResult`; la ayuda del verbo no cambia |
| `cmd/axiom/main.go` | Modificar | `svc.SetMarksPublisher(cli.NewMarksPublisher())` |
| `internal/dashboard/marks_port.go` | Crear | Interfaz `MarksPublisher` y `SetMarksPublisher` |
| `internal/dashboard/service_marks.go` | Crear | Disponibilidad, cerrojo, `RecordMark`, `PublishMarks`, DTO de marcas |
| `internal/dashboard/service.go` | Modificar | Campo `marks`/`marksLocks`, reloj inyectable; `incrementAcceptanceTests` rellena `Marks` |
| `internal/dashboard/types.go` | Modificar | `AcceptanceMarksDTO` y anexos |
| `internal/dashboard/server.go` | Modificar | Dos `HandleFunc` y sus manejadores |
| `internal/dashboard/secrets_canary_test.go` | Modificar | Dos entradas en `excludedRoutes` |
| `internal/dashboard/assets/js/screens/detalle-pruebas.js` | Modificar | Matriz de marcas y publicación (D15); se actualiza su comentario de cabecera |
| `internal/assets/skills/sdd-archive/SKILL.md` | Modificar | Paso 2c «Marcas de pruebas» y sección del informe |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | Resumen de marcas en `integration` y en el resumen final de `continuous` |
| `internal/assets/skills/_shared/sdd-status-contract.md` | Modificar | Documenta `acceptanceTests.marks` |
| `testdata/golden/*` | Regenerar | Los que incluyen `sdd-archive` y la doctrina del orquestador |

## Interfaces y contratos

```go
// internal/acceptancetests/marks
const (
	FileName       = "pruebas-estado.yaml"
	MaxNoteRunes   = 500
	MaxCommitRunes = 100
)
type Result string // "ok" | "ko" | "blocked" | "n/a"
type Entry struct {
	Case        string `yaml:"case"`
	Environment string `yaml:"environment"`
	Result      Result `yaml:"result"`
	By          string `yaml:"by"`
	At          string `yaml:"at"`
	Commit      string `yaml:"commit"`
	PlanDigest  string `yaml:"plan_digest"`
	Note        string `yaml:"note"`
}
type File struct{ Entries []Entry; Invalid []Issue }
func Parse(content []byte) (File, error)
func Append(content []byte, e Entry) ([]byte, error)
func Merge(local, remote []byte) ([]byte, error)
func Pending(local, published []Entry) []Entry
type Input struct{ Case, Environment, Result, Commit, Note string }
type Rejection struct{ Code, Message, Continuation string }
func ValidateInput(in Input, battery acceptancetests.Battery, environments []string) (Entry, *Rejection) // sin By, At ni PlanDigest
func Summarize(battery acceptancetests.Battery, environments []string, entries []Entry, digest string) Summary
func (s Summary) Text() string

// internal/sddstatus
func AcceptanceMarksFor(ctx context.Context, location workspace.Location, changeRoot, change string, tests AcceptanceTestsResult) AcceptanceMarksResult

// internal/cli
type MarksPublishRequest struct{ CWD, Change, Branch string }
type MarksPublishResult struct {
	Status, Remote, Branch, GitRoot, Commit, PushedSHA string
	Replayed        bool
	UncommittedDocs []string
	Refusal         *MarksPublishRefusal // Code, Label, Message, Continuation
}
func NewMarksPublisher() *MarksPublisher
func (*MarksPublisher) MarksIdentity(ctx context.Context, dir string) (string, []string)
func (*MarksPublisher) PublishMarks(ctx context.Context, req MarksPublishRequest) MarksPublishResult

// internal/sddpublish (puerto Git)
Fetch(ctx context.Context, dir, remote, branch string) (PushOutcome, error)
FileAt(ctx context.Context, dir, rev, rel string) (content []byte, exists bool, err error)
ResetKeep(ctx context.Context, dir, rev string) error
```

## Doctrina y goldens

- **`sdd-archive/SKILL.md`**: nuevo «Step 2c: Acceptance Marks Summary (before Step 3)». Antes de mover la carpeta, leer `acceptanceTests.marks` de `axiom sdd status --cwd <root> <change> --json`. Con `summary`, el `archive-report` incluye `## Marcas de pruebas` con el texto literal; con `pending > 0`, la línea «Aviso: <n> marcas sin publicar; viajan en la publicación del archivado» (en multirepo el archivado es in situ y `axiom sdd publish --archive` confirma la carpeta archivada entera, fichero incluido); con `error`, un aviso de fichero ilegible; sin la clave, sin sección. Nunca bloquea el archivado; el fichero se mueve intacto y la comparación `diff -r` existente lo cubre (prov:ARC:1, hueco 8).
- **`_shared/sdd-orchestrator-sections.md`**: junto al párrafo de la verificación conjunta en `integration`, una frase: con `acceptanceTests.marks` proyectado, la presentación muestra su `summary` literal y, con `pending > 0`, cuántas marcas faltan por publicar; es información que no cambia las dos salidas ni la evidencia de fusión, no se registra en `gates.yaml`, y un `ko`, `blocked`, sin marcar o caducado nunca se presenta como limpio; en `continuous` va al resumen final (prov:GAT:1).
- **`_shared/sdd-status-contract.md`**: amplía la viñeta de `acceptanceTests` con `marks`.
- Goldens: `go test ./internal/components -run TestGolden -update` y, después, sin `-update` para comprobar.

## Estrategia de pruebas

Todas localizadas por paquete y con `-run`; sin binarios compilados; remotos desnudos locales en proceso. Las pruebas del dashboard que usan `freePort` o el hub no llevan `t.Parallel`.

| Capa | Qué | Cómo |
|---|---|---|
| Unidad | `marks`: parse válido/ilegible/no secuencia, entradas inválidas, `Append` prefijo exacto, ida y vuelta hostil, `Merge` sin duplicados (también entradas idénticas ya en el remoto), `Pending` multiconjunto, `Summarize` (n/a, ko, blocked, sin marca, caducada, varias vigentes, total 0, huérfanas, CRLF), `Text`, validación de cada campo, importaciones | `go test ./internal/acceptancetests/marks` |
| Unidad | `ExactAllowed` acepta solo `pruebas-estado.yaml` y rechaza `design.md`, `../x`, rutas absolutas, `sub/pruebas-estado.yaml` | `go test ./internal/docpaths -run TestExactAllowed` |
| Integración (git real) | `Only`: solo ese fichero en el commit; otros documentos sucios y preparados intactos en árbol e índice y avisados; `wrong_branch`; `identity_missing`; `remote_missing`; `other_commits_unpublished`; `only_path_rejected` y `only_requires_in_place` sin ninguna llamada git; reintento con éxito (segundo clon sube antes); hook `pre-push` sin `fetch`; `axiom sdd publish` sin `fetch` ni reintento | `go test ./internal/sddpublish -run 'TestOnly\|TestReplay\|TestIsolation'` |
| Unidad (puerto falso) | Doble `non-fast-forward` → sin tercer intento y commit conservado; fallos de R1–R5 dejan el estado descrito en D10 | mismo paquete, `-run TestReplay` |
| Integración | `PublishedChangeFile` con y sin referencia remota y fichero | `go test ./internal/branch/engine -run TestPublishedChangeFile` |
| Integración | Proyección `marks` (multirepo), ausencia en monorepo y sin fichero, `nextRecommended`/dependencias/digests idénticos con y sin fichero, `pending` y `pendingUnavailable` | `go test ./internal/sddstatus -run TestAcceptanceMarks` |
| Integración | Rutas: cada código de D13 y D14, sin escritura en rechazos, solo añadir, 2 marcas concurrentes, `marks_busy`, publicar sin `confirm` sin llamar al puerto, `nothing_to_publish`, DTO del detalle = status, nota hostil literal; canario | `go test ./internal/dashboard -run 'TestMarks\|TestEveryDashboardRouteIsClassifiedForTheSecretsCanary'` con `MarksPublisher` falso |
| Arquitectura | Los ficheros de marcas del dashboard no importan `os/exec`, `gitcommon`, `gitidentity` ni `sddpublish`; aislamiento de `sddpublish` y único punto de ejecución | `go test ./internal/dashboard -run TestMarksFilesRunNoGit`, `./internal/sddpublish -run TestIsolation`, `./internal/gitcommon -run TestArchitecture` |
| Integración | `MarksPublisher` contra repo multirepo de fixture con remoto local; identidad ausente; trinquete de rechazos | `go test ./internal/cli -run 'TestMarksPublisher\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign'` |
| Assets | `detalle-pruebas.js` escapa notas y `commit`, envía `confirm: true` y `branch` solo tras `confirmDialog`, no ofrece control en celdas no aplicables | `go test ./internal/dashboard -run TestAcceptanceMarksAssets` |
| Doctrina | Goldens | `go test ./internal/components -run TestGolden` |

Los fixtures nuevos usan IDs `RF-` (nunca `REQ-`), así que `internal/idmigration/residues.yaml` no cambia. El trinquete de código muerto (CI) exige que todo lo exportado tenga uso real: `marks.*` lo usan `sddstatus`, `dashboard` y `cli`; `PublishedChangeFile`, `sddstatus`; `NewMarksPublisher` y `SetMarksPublisher`, `cmd/axiom`.

## Matriz de amenazas

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta de diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas tipo documentación | `design.md`, `internal/x.go`, `../pruebas-estado.yaml`, ruta absoluta, `sub/pruebas-estado.yaml` en `Only` | Applicable | `ExactAllowed` puro antes de git; solo el nombre exacto en la raíz del cambio | Un caso por clase en `docpaths` y `only_path_rejected` sin llamadas git en `sddpublish` |
| Selección del repositorio git | worktree registrado obsoleto, `cwd` del proceso, destino no `InPlace` | Applicable | `dir` siempre `PublicationTarget.GitRoot`; marcas en `PlanningHome` in situ (D6); `only_requires_in_place` | Worktree obsoleto → commit en el repo de specs; destino no in situ → rechazo |
| Estado del commit | otro documento preparado, otro sucio sin preparar, nada sucio con commit de marcas sin subir | Applicable | `commit --only -- <ruta>`; avisos en `UncommittedDocs`; rango solo de marcas | Índice y árbol de los otros intactos; push sin commit nuevo |
| Estado del push | sin referencia remota, `non-fast-forward` una y dos veces, hook `pre-push`, refspec explícita | Applicable | Refspec `refs/heads/<b>:refs/heads/<b>` sin `+`, sin `--set-upstream` in situ; un único `fetch` + reproducción (D10) | Reintento con éxito; doble rechazo sin tercer intento; hook sin `fetch`; publicación ordinaria sin `fetch` |
| Comandos de PR | — | N/A: el cambio no abre ni compone PR | — | — |

## Migración y despliegue

No requiere migración. Es aditivo: sin claves nuevas en `axiom.yaml` (se reutiliza `acceptance_tests.environments`), sin cambios en el JSON de `axiom sdd publish` ni en el de status cuando no hay `pruebas-estado.yaml`.

## Huecos de la revisión de cobertura y su decisión

| # | Hueco | Decisión |
|---|---|---|
| 1 | Qué es «pendiente» y no duplicar al reproducir | D3: pendientes = local − versión en `refs/remotes/origin/<rama>` (multiconjunto por los ocho campos); `Merge` añade solo lo que falta en el remoto |
| 2 | Marca sin identidad git | D5: rechazo `identity_missing` (409) con las claves y órdenes; nada escrito |
| 3 | Validación y serialización de `commit` y `note` | D5: `note` ≤ 500 runas sin controles; `commit` ≤ 100 runas y juego de caracteres cerrado; codificación solo con `yaml.v3`, ida y vuelta probada |
| 4 | Semántica del resumen | D4: categorías disjuntas, `n/a` en el total y aparte, vigente por `at`, total 0 = «sin casos», huérfanas e inválidas aparte, texto canónico único |
| 5 | Fallos y estado local del reintento | D10: tabla R1–R6; todo fallo antes de `reset --keep` deja el estado intacto; commit conservado; recuperación por `ORIG_HEAD` |
| 6 | Cerrojo marca/publicación | D7: cerrojo por cambio; marcas serializadas; publicación con `TryLock` y `marks_busy` |
| 7 | Copia donde se escribe y lee | D6: `PlanningHome` in situ, nunca `worktreePath`; prueba de igualdad DTO = status |
| 8 | Marcas sin publicar al archivar | Doctrina: aviso informativo en el `archive-report` con `pending`; en multirepo viajan en `axiom sdd publish --archive` |

## Unidades de trabajo sugeridas (para sdd-tasks)

| Orden | Unidad | Depende de | Paquetes de prueba |
|---|---|---|---|
| U1 | Núcleo `internal/acceptancetests/marks` + `docpaths.ExactAllowed` | — | `marks`, `docpaths` |
| U2 | `Engine.PublishedChangeFile` + `sddstatus.AcceptanceMarksFor` + proyección `acceptanceTests.marks` + contrato de status | U1 | `branch/engine`, `sddstatus` |
| U3 | `sddpublish`: `Only`, `Replay`, puerto (`Fetch`, `FileAt`, `ResetKeep`), códigos nuevos | U1 (fusión de prueba) | `sddpublish`, `gitcommon` |
| U4 | Dashboard: puerto, disponibilidad, cerrojo, rutas, DTO, canario (con publicador falso) | U1, U2 | `dashboard` |
| U5 | `cli.MarksPublisher` + inyección en `cmd/axiom/main.go` | U3, U4 | `cli` |
| U6 | UI `detalle-pruebas.js` + prueba de assets | U4 | `dashboard` |
| U7 | Doctrina (`sdd-archive`, orquestador) + goldens | U2 | `components` |

Cada unidad cierra con su commit; las exportaciones de U1–U4 tienen su uso real al final de U5, que es cuando el trinquete de código muerto (CI) las evalúa.

## Preguntas abiertas

- Ninguna bloqueante. Se asume que un único dashboard por máquina escribe marcas (D7) y que la ausencia de token CSRF sigue fuera de alcance (riesgo heredado de la propuesta).
