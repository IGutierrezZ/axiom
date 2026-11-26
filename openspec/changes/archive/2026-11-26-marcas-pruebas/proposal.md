# Propuesta: Marcas de ejecución por entorno de `pruebas.md`, publicadas a la integración del repo de specs (C-06-d)

## Intención

Hoy la batería de `pruebas.md` (C-06-c) es un plan congelado por la puerta `tasks`, pero no hay forma de registrar qué casos se han ejecutado en cada entorno ni con qué resultado. Los equipos llevan ese seguimiento fuera de Axiom y la puerta `integration` no lo ve.

Este cambio permite que una persona marque en la UI, por caso y entorno, el resultado de la ejecución (`ok`, `ko`, `blocked`, `n/a`) y que esas marcas se compartan con todo el equipo publicándolas con push directo a la rama de integración del repo de specs, que es la fuente compartida (decisión de usuario del 2026-11-13: «OpenSpec/ODD en git es la fuente compartida suficiente»). El resumen por entorno («pre: 5/5 ok») se proyecta en `axiom sdd status`, en la presentación de `integration` y en el `archive-report`, siempre como información y nunca como bloqueo.

Éxito: una persona marca un caso, ve la entrada añadida en local, pulsa «Publicar marcas», confirma, y el commit de `pruebas-estado.yaml` llega a la integración del repo de specs; si `pruebas.md` cambia, las marcas antiguas aparecen caducadas.

## Alcance

### Dentro del alcance

- **Artefacto de estado `pruebas-estado.yaml`** en la carpeta del cambio, separado de `pruebas.md`, de solo añadir. Cada entrada: `case` (P-NN), `environment`, `result` (`ok|ko|blocked|n/a`), `by` (identidad git `user.name`/`user.email` del repo de specs), `at` (fecha y hora), `commit` (opcional, lo escribe la persona; vacío por defecto, nunca se infiere del repo de código), `plan_digest` y `note` (acotada).
- **Caducidad:** `plan_digest` = `kickoff.ArtifactDigest` de `pruebas.md` (mismo digest que la puerta `tasks`, con CRLF normalizado). Una marca con digest distinto del actual se muestra como caducada.
- **Validación en servidor** de cada marca: `case` existe en `pruebas.md`; `environment` pertenece a `acceptance_tests.environments` y a los entornos aplicables del caso; `result` pertenece al dominio cerrado; `note` con longitud máxima. Las marcas desde la UI son humanas.
- **Pantalla de marcas** en el detalle del incremento: tabla de casos con una columna por entorno, celdas con resultado y nota, estado caducado y lista de entradas pendientes de publicar.
- **Flujo en dos pasos:**
  1. `POST` de marca: solo escribe en local (añade la entrada con escritura atómica y exclusión mutua por cambio). Sin git.
  2. `POST` «Publicar marcas»: exige confirmación explícita (clic más campo `confirm` en el cuerpo). Antes del clic la UI muestra remoto, rama y entradas pendientes. Nunca hay push sin ese clic.
  Ambas rutas protegidas con `isLocalRequest` y clasificadas en `excludedRoutes` del canario de secretos («mutación (POST)»).
- **Publicador de marcas acotado (enfoque A):** interfaz propia del dashboard, implementada en `internal/cli` e inyectada desde `cmd/axiom`. Reutiliza `sddpublish` con una lista exacta de rutas (`Request.Only`) limitada a `pruebas-estado.yaml` del cambio y exige `PublicationTarget.InPlace` sobre la rama de integración del repo de specs (ignora cualquier `worktreePath` registrado). Ante `push_non_fast_forward`, y solo en este modo, un único reintento por reproducción: `fetch` del remoto, descartar el commit local de marcas, avance rápido, volver a añadir las entradas pendientes, commit y push. Si vuelve a fallar, rechazo tipado claro con su continuación.
- **Otros documentos sucios** en el repo de specs no bloquean: la publicación de marcas commitea solo `pruebas-estado.yaml`, deja el resto intacto e informa de ello como aviso.
- **Resumen por entorno** (casos `ok`/total vigentes, caducados aparte) en:
  - la proyección `acceptanceTests` de `axiom sdd status` (aditiva);
  - la presentación de la puerta `integration` (doctrina del orquestador y `sdd-archive`);
  - el `archive-report`.
  Siempre informativo: no toca `nextRecommended`, dependencias, motivos de bloqueo ni digests, y no bloquea el archivado.
- **Topología:** solo multirepo y solo SDD. En monorepo la pantalla y las rutas responden con un estado «no disponible» explícito (no 404). Cambio archivado o sin `pruebas.md`: sin marcas. Sin claves nuevas en `axiom.yaml` (se reutiliza `acceptance_tests.environments`).

### Fuera del alcance

- Monorepo y carril ODD.
- **Vía de escritura de agentes.** La regla «un agente solo registra evidencia automatizada en `local`» queda fuera de v1: no se añade ningún verbo de CLI ni ruta para agentes; todas las marcas v1 son humanas desde la UI.
- Sincronizar marcas con Azure DevOps Test Plans u otro gestor.
- Bloquear el archivado, una puerta o `nextRecommended` por marcas.
- Que la UI publique en ramas de cambio o en worktrees.
- Inferir el commit probado del repo de código.
- Editar o borrar marcas ya escritas (el fichero solo se añade; una corrección es una entrada nueva).
- Marcas sobre cambios archivados.
- Token CSRF en el dashboard (deuda transversal; ver riesgos).
- Reintento con `rebase` o cualquier reintento en `axiom sdd publish` ordinario.

## Capacidades

### Capacidades nuevas

Ninguna.

### Capacidades modificadas

- `local-web-dashboard`: **modificar RF-WEB-17** (la pestaña «Pruebas» deja de ser de solo lectura y sin ruta nueva: añade marcas por entorno y dos rutas POST). **Añadir** requisitos `prov:WEB:n` para: pantalla de marcas y caducidad; ruta de marca (validación en servidor, escritura local de solo añadir); ruta de publicación (confirmación explícita, datos visibles antes del clic, resultado o rechazo tipado); disponibilidad solo en multirepo SDD con estado «no disponible» en monorepo; clasificación en el canario de secretos.
- `sdd-docs-publication`: **modificar RF-PUB-01** (la lista cerrada admite un modo de lista exacta limitado a `pruebas-estado.yaml`), **RF-PUB-05** (excepción acotada: solo el publicador de marcas, ante *non-fast-forward*, hace un único reintento por reproducción con `fetch` autorizado por el mismo clic; `axiom sdd publish` sigue sin reintentar) y **RF-PUB-07** (el puerto de publicación también lo alcanza el publicador de marcas, inyectado desde `internal/cli`; el dashboard no importa `sddpublish` ni ejecuta git). **Añadir** `prov:PUB:n` para la publicación de marcas (destino `InPlace` en la integración del repo de specs, otros documentos sucios como aviso, rechazo tipado tras el reintento).
- `sdd-lane-kickoff`: **modificar RF-KOF-22** (la proyección `acceptanceTests` añade el resumen de marcas por entorno y las marcas caducadas, de forma aditiva y sin efectos sobre el enrutado). **Añadir** `prov:KOF:n` para el formato de `pruebas-estado.yaml` y su caducidad por `plan_digest`.
- `sdd-block-review-gates`: **añadir** `prov:GAT:n`: la presentación de `integration` muestra el resumen de marcas como evidencia informativa, no bloqueante, sin cambiar sus dos salidas ni la evidencia de fusión exigida (complementa RF-GAT-26 sin modificarlo).
- `sdd-archive-lifecycle`: **añadir** `prov:ARC:n`: el `archive-report` incluye el resumen de marcas y las marcas nunca bloquean el archivado.

Nota: el orquestador citó tres capacidades; `sdd-lane-kickoff` (dueña de la proyección `acceptanceTests`, RF-KOF-22) y `sdd-archive-lifecycle` (dueña del archivado) se añaden porque el resumen en status y en el `archive-report` es comportamiento de spec que vive allí.

## Enfoque

Enfoque A de la exploración: el dashboard define una interfaz `MarksPublisher` propia; `internal/cli` la implementa sobre `sddpublish` y `cmd/axiom/main.go` la inyecta al construir el servicio. Así se conserva el aislamiento de RF-PUB-07 (un único punto de commit/push en `sddpublish/gitport.go`) y se reutilizan los rechazos tipados y la clasificación de push existentes.

`sddpublish` gana un modo acotado: `Request.Only` (lista exacta de rutas, validada contra la lista cerrada) y el reintento por reproducción, activado solo junto con `Only` desde el publicador de marcas. El reintento reproduce las entradas en vez de hacer `rebase`, lo que evita conflictos de texto en un YAML de solo añadir. La lectura de identidad (`user.name`/`user.email`) se añade como lectura mínima en el puerto de git.

La escritura local de marcas vive en el servicio del dashboard (lectura de `pruebas.md`, validación, `fileprim` atómico, mutex por cambio). El resumen y la caducidad se calculan en `sddstatus` (extensión de `AcceptanceTestsProjection`) y el dashboard los consume del mismo origen, para que status, UI y presentación de `integration` coincidan.

Se descarta B (subproceso `axiom sdd publish --only`): depende del binario en ejecución, obliga a parsear JSON y exige el mismo cambio en `Request`, con peor testabilidad.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/sddpublish/` (`publish.go`, `ports.go`, `gitport.go`, `result.go`, `classify.go`) | Modificado | `Request.Only`, reintento por reproducción acotado, lectura de identidad, rechazo tipado tras reintento |
| `internal/docpaths/` | Modificado | Validación de una lista exacta contra la lista cerrada |
| `internal/cli/sdd_publish.go` (o fichero hermano) | Modificado | Implementación del publicador de marcas; ajuste de `TestIsolationOnlyTheCLIVerbImportsSddpublish` si cambia de fichero |
| `cmd/axiom/main.go` | Modificado | Inyección del publicador en el servicio del dashboard |
| `internal/dashboard/server.go` | Modificado | Dos rutas POST con `isLocalRequest`; estado «no disponible» |
| `internal/dashboard/service.go`, `service_listing.go`, `types.go` | Modificado | Escritura de marcas, validación, DTO de marcas y pendientes |
| `internal/dashboard/assets/js/screens/` (`detalle-pruebas.js`, `detalle.js`) | Modificado | Celdas de marca, botón «Publicar marcas» con confirmación |
| `internal/dashboard/secrets_canary_test.go` | Modificado | Clasificación de las rutas nuevas |
| `internal/sddstatus/acceptance_tests.go` | Modificado | Resumen por entorno y caducidad en la proyección |
| `internal/acceptancetests/` | Modificado | Lectura y modelo de `pruebas-estado.yaml` (si encaja aquí; lo decide el diseño) |
| `internal/assets/skills/sdd-archive/SKILL.md`, `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, `_shared/sdd-status-contract.md` | Modificado | Resumen en `integration`, en el `archive-report` y en el contrato de status |
| `testdata/golden/*` (sdd-archive, doctrina) | Modificado | Regeneración de goldens |
| `openspec/specs/{local-web-dashboard,sdd-docs-publication,sdd-lane-kickoff,sdd-block-review-gates,sdd-archive-lifecycle}` | Modificado | Deltas (al archivar) |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| Push desde la UI sin intención real | Media | Dos pasos; publicar exige clic y `confirm` en el cuerpo; la UI muestra antes remoto, rama y pendientes; marcar nunca toca git |
| **Ausencia de token CSRF** en el dashboard: `isLocalRequest` solo comprueba Host y Origin | Media | Usar `isLocalRequest` en ambas rutas más `confirm` obligatorio y cuerpo JSON (no formulario simple); dejar el CSRF como deuda explícita fuera de alcance |
| El reintento con `fetch` contradice el «nunca fetch» de RF-PUB-05 | Media | Excepción escrita en el delta, limitada al modo `Only` del publicador de marcas, un único intento y autorizada por el mismo clic; `axiom sdd publish` sin cambios |
| Publicación concurrente de dos personas | Media | Solo añadir + reproducción de entradas; si el segundo intento falla, rechazo tipado claro y el fichero local conserva las entradas |
| Arrastrar otros documentos sucios del repo de specs | Baja | `Only` commitea únicamente `pruebas-estado.yaml`; el resto queda intacto y se avisa |
| Leer de un worktree retirado en vez de la integración | Media | Ignorar `worktreePath` al escribir; exigir `PublicationTarget.InPlace` |
| Caducidad espuria por finales de línea | Baja | `kickoff.ArtifactDigest` normaliza CRLF |
| **Solape con la ola 13** en `server.go`, `service.go`, goldens y `sdd-archive` (README §4) | Alta | Cambios aditivos y localizados; regenerar goldens al final; resolver al fusionar |
| Trinquetes (código muerto, resoluciones de rechazo, canario de secretos, aislamiento) | Media | Todo lo exportado con uso real desde `cmd/axiom`/`cli`; cada rechazo nuevo con continuación; rutas clasificadas |
| Este repositorio es `monorepo-embedded`: el flujo real multirepo no se ejercita aquí | Media | Pruebas con fixtures multirepo y remoto local en proceso (sin binarios sueltos) |

## Plan de reversión

- El cambio es aditivo: revertir los commits de la rama `inc/marcas-pruebas` (o de su PR) elimina rutas, pantalla, modo `Only`, reintento y resumen sin migración de datos.
- Los `pruebas-estado.yaml` ya publicados en repos de specs son ficheros de documentación inertes: tras revertir, nadie los lee y pueden conservarse o borrarse a mano; no afectan a puertas, digests ni archivado.
- `axiom sdd publish` ordinario no cambia de comportamiento, así que revertir no altera la publicación de documentos existente.
- Los goldens se regeneran de nuevo tras revertir el texto de `sdd-archive` y de la doctrina.

## Dependencias

- C-06-c (`pruebas.md`, `acceptance_tests.environments`, `acceptancetests.Check`, proyección `acceptanceTests`), ya integrado.
- `axiom sdd publish` (`internal/sddpublish`) y `Engine.PublicationTarget` con la rama de integración del repo de specs in situ (secciones 02 y 05).
- Coordinación con los cambios de la ola 13 que tocan `server.go`, `service.go`, goldens y `sdd-archive`.

## Criterios de éxito

- [ ] Desde la UI se marca un caso en un entorno y `pruebas-estado.yaml` recibe una entrada con `by` (identidad git), `at`, `commit` (opcional, vacío por defecto) y `plan_digest`, sin ninguna operación git.
- [ ] Una marca con `case`, `environment`, `result` o `note` inválidos se rechaza en el servidor sin escribir nada.
- [ ] «Publicar marcas» sin `confirm` no ejecuta git; con `confirm`, commitea solo `pruebas-estado.yaml` y hace push a la integración del repo de specs, mostrando el resultado o el rechazo tipado.
- [ ] Ante *non-fast-forward*, el publicador de marcas reintenta una sola vez por reproducción; si falla, devuelve un rechazo tipado con continuación. `axiom sdd publish` sigue sin reintentar.
- [ ] El dashboard no importa `sddpublish` y la prueba de aislamiento sigue en verde.
- [ ] Al cambiar `pruebas.md`, las marcas con `plan_digest` antiguo aparecen caducadas.
- [ ] `axiom sdd status`, la presentación de `integration` y el `archive-report` muestran el resumen por entorno sin alterar `nextRecommended`, dependencias ni digests; el archivado no se bloquea por marcas.
- [ ] En monorepo la pantalla y las rutas responden «no disponible»; el canario de secretos y las pruebas de pines siguen en verde.
