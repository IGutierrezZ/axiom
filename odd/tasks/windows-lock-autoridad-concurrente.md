# ODD: fallo intermitente del bloqueo de autoridad con publicadores concurrentes en Windows (H8)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/windows-lock-autoridad-concurrente.md`, en el repositorio `axiom`. La rama y el worktree dedicados se crean al iniciar la ejecución, nunca en `C:\repos\axiom`.
> Espejo de recuperación en Engram: topic `odd/windows-lock-autoridad-concurrente/tasks`, proyecto `axiom` (creado el 2026-10-13 en estado propuesto; se actualiza al iniciar la ejecución).
> **Estado: Propuesto; no iniciado. Se ejecuta en una sesión aparte.**
> Origen: hallazgo H8 del ODD `windows-pruebas-entorno-y-ci` (`odd/tasks/windows-pruebas-entorno-y-ci.md`).

## Objetivo

Saber si el fallo de `TestRARPlanAuthorityConcurrentExactReplayConverges` en Windows es contención real o un defecto de equidad del bloqueo, o si es solo falta de holgura del runner. Dejar la prueba estable en Windows sin ocultar la causa, y documentar esa causa.

## Problema y porqué

- `TestRARPlanAuthorityConcurrentExactReplayConverges` (`internal/reviewtransaction/rar_plan_authority_test.go:82`) lanza 8 publicadores concurrentes de `PublishPlan` sobre el mismo plan y exige que todos converjan en la misma autoridad.
- En el runner de Windows falló: 7 de los 8 publicadores recibieron `authority lock acquisition timed out after 2s` (`ErrAuthorityLockTimeout`).
- La «divergencia» que informa la prueba es una consecuencia: compara la autoridad de una llamada que falló, que es el valor cero, con la del ganador.
- Es la única prueba en rojo de una línea base en la que 9 de 10 shards pasaron. Ha ocurrido una sola vez, así que no hay todavía una tasa de fallo.
- Hasta saber la causa, cualquier retoque sería una apuesta. Un reintento o un presupuesto mayor sin diagnóstico podría ocultar un defecto real del bloqueo, que es código de producción de la autoridad de revisión.

## Evidencia

- Run `38289022694`, shard `reviewtransaction-l-z`, `workflow_dispatch` sobre `main` `d47d51aa`, 2026-10-12.
- Qué bloqueo golpea la prueba (verificado en el árbol actual):
  - `PublishPlan` (`rar_plan_authority.go:322`) toma el bloqueo en `rar_plan_authority.go:396` con `acquireRARAuthorityLock` (`rar_path_safety.go:360`).
  - Este llama a `acquireBoundedRARLock` (`rar_path_safety.go:467`), que espera como máximo `maintenanceLockTimeout`, sondeando cada 10 ms.
  - `maintenanceLockTimeout` es una **constante** de 2 s (`store_lock.go:25`). Desde una prueba no se puede escalar.
  - `compactStartLockTimeout` (`compact_store.go:39`) es otro presupuesto de 2 s, pero es una variable y solo lo usa `acquireCompactStartLock` (`compact_store.go:1101`), el inicio de la transacción compacta. **No** es el bloqueo de esta prueba.
- Rama de convergencia: ante un agotamiento del bloqueo, `PublishPlan` devuelve la autoridad calculada si el objeto inmutable ya existe en disco con los mismos bytes y la instantánea sigue viva (`rar_plan_authority.go:397-421`). Si el ganador aún no ha publicado el objeto, devuelve el error de tiempo.
- La sección crítica incluye la validación de la instantánea viva y de los contratos nativos (`validateLivePlanSnapshot`, `validateNativePlanContracts`) antes de publicar, y esas validaciones consultan git. Esto es una **hipótesis de trabajo**, no una conclusión: hay que medirlo.
- La prueba hermana `TestRARPlanAuthorityConvergesOnExhaustedLockOverIdenticalPublishedAuthority` cubre de forma determinista la forma de agotamiento en la que el ganador ya publicó. Aquí el ganador no había publicado cuando los demás agotaron la espera.

## Alcance autorizado

El alcance previsto es investigar y, según el diagnóstico, uno de estos dos resultados. **El usuario debe autorizar la implementación al iniciar la sesión**; hasta entonces el trabajo es de solo lectura y de medición.

- Reproducir y diagnosticar (T1 y T2): lectura, ejecuciones locales focalizadas y, con permiso, una ejecución en el runner de Windows.
- Si es un defecto real del bloqueo: corregirlo en el producto, con una decisión explícita del usuario sobre el alcance.
- Si es falta de holgura: ajustar la prueba (número de publicadores o presupuesto del lado de la prueba con `internal/testenv`).
- **Fuera de alcance:** cambiar los presupuestos de producción (`maintenanceLockTimeout`, `compactStartLockTimeout`) sin una decisión explícita del usuario; reintentos que enmascaren el fallo; el resto de pruebas de `reviewtransaction`.

## Restricciones

- Commits en castellano, Conventional Commits, **sin `Co-Authored-By` ni atribución de IA**.
- Trabajar en un worktree dedicado; `C:\repos\axiom` es el checkout de otra sesión y no se toca.
- En Windows, en local solo paquetes focalizados con `-run`, nunca la suite completa.
- Push, PR y merge solo con permiso del usuario. Lanzar el runner de Windows cuesta minutos de CI y también necesita permiso.
- **Sin reintentos** para enmascarar un fallo. Una prueba que falla se explica, no se repite hasta que pase.
- Los presupuestos de producción no cambian sin decisión explícita del usuario.

## Modo de trabajo

Re-comprobar al iniciar.

- **TDD:** modo estándar. Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test`.
- **RDD:** releer el estado con `axiom review mode status` y no asumir el valor del ODD de origen. Si está activo, la tarea tocará código de la autoridad de revisión y su riesgo probable es alto.
- **Entrega:** estrategia `ask-on-risk`. Previsión orientativa: 0 líneas de producción si el resultado es solo documentar; de 40 a 120 líneas de autoría si se ajusta la prueba o se corrige el bloqueo, con sus pruebas. Si se supera con claridad el umbral de unas 400 líneas, se pregunta una vez por la estrategia de cadena.

## Tareas

- [ ] **T1 · Reproducir**
  - Ruta: en línea (comandos).
  - Local, primero y gratis: `go test ./internal/reviewtransaction -run '^TestRARPlanAuthorityConcurrentExactReplayConverges$' -count=20`. Anotar los fallos, los tiempos y la carga de la máquina.
  - En el runner de Windows, **con permiso del usuario** y avisando de que cuesta minutos de CI: ejecutar la misma prueba con `-count=20`. Elegir la opción menos invasiva entre dos:
    - un `workflow_dispatch` con un input temporal de patrón de prueba, en una rama que no se fusiona;
    - una rama con la etiqueta `ci:windows`, que lanza `windows-full-suite` entera. Es más cara: preferir la primera si el input es pequeño.
  - Comprobación: tasa de fallo observada (n fallos de 20) en local y en el runner. Sin commit; el resultado va a este documento.
- [ ] **T2 · Diagnosticar**
  - Ruta: delegada (mapeo de 4 o más ficheros: `store_lock.go`, `rar_path_safety.go`, `rar_plan_authority.go`, `rar_authority_repository.go` y el bloqueo por plataforma). Disparador: mapeo de más de 4 ficheros.
  - Preguntas a responder con medición:
    - ¿Cuánto dura la sección crítica del ganador en el runner de Windows? ¿La dominan las consultas a git de la validación?
    - ¿Sirve el bloqueo de forma equitativa o puede un publicador esperar varios sondeos de 10 ms sin ganar nunca? (`LockFileEx` no garantiza el orden.)
    - ¿Por qué no había publicado el ganador cuando el resto agotó la espera? ¿Con 8 publicadores serializados el total supera los 2 s?
  - Resultado: una de dos conclusiones, **contención real o defecto de equidad** del bloqueo, o **falta de holgura** de Windows para 8 publicadores. Si no se puede decidir con los datos, se dice, y se pide más ejecuciones en lugar de elegir.
  - Sin commit; el diagnóstico va a este documento.
- [ ] **T3 · Decidir con el usuario**
  - Ruta: el padre, con una pregunta bloqueante. Presentar el diagnóstico y las opciones con su coste.
  - Si es un defecto real: pasa a ser un arreglo de producto, con su propio alcance y su prueba determinista de la forma del fallo.
  - Si es falta de holgura: se escala el presupuesto del lado de la prueba con `internal/testenv` (`AXIOM_TEST_TIMEOUT_SCALE`) **solo si** el presupuesto es alcanzable desde las pruebas, y hoy no lo es (`maintenanceLockTimeout` es una constante). Entonces la opción de prueba es reducir el número de publicadores o cambiar la constante por una variable de pruebas, y esto último toca código de producción.
  - Los presupuestos de producción no cambian sin decisión explícita del usuario.
  - Decisión registrada aquí antes de escribir código.
- [ ] **T4 · Implementar**
  - Solo con la decisión de T3 y la autorización del usuario. Ruta según el tamaño: en línea si es un fichero mecánico; escritor delegado si son dos o más ficheros no triviales.
  - Primero una prueba que falle por la causa diagnosticada (RED) cuando el caso sea un defecto real.
  - Comprobación:
    - local, focalizada: `go test ./internal/reviewtransaction -run 'TestRARPlanAuthority|TestMaintenanceLock' -count=1`;
    - runner de Windows, con permiso: **20 de 20** con `-count=20` sobre `^TestRARPlanAuthorityConcurrentExactReplayConverges$`.
  - Commit (conforme al resultado): `fix(reviewtransaction): <causa diagnosticada>` o `test(reviewtransaction): <ajuste de la prueba concurrente en Windows>`.
- [ ] **T5 · Cierre**
  - `go build ./...`, `go vet ./...`, `go run ./internal/gofmtcheck`, `bash scripts/deadcode-ratchet.sh`, los paquetes tocados con `-count=1` y `git status --porcelain` vacío.
  - Si RDD está activo, evaluar el candidato con `axiom review assess` antes de declarar el trabajo terminado.
  - Actualizar este documento y el espejo. Preparar el PR; push, PR y merge los decide el usuario.

## Criterios de aceptación

- 20 de 20 ejecuciones correctas en el runner de Windows de `TestRARPlanAuthorityConcurrentExactReplayConverges`.
- La causa raíz está documentada con datos medidos, no inferida.
- Ningún fallo queda enmascarado: sin reintentos, sin presupuestos de producción cambiados a ciegas, sin omisiones de la prueba.
- Si la causa es un defecto real, queda corregido con una prueba determinista que reproduce su forma.

## Progreso

- 2026-10-13: documento abierto desde el cierre del ODD `windows-pruebas-entorno-y-ci`. Verificado en el árbol de `main` `eea58801` qué bloqueo usa la prueba. No iniciado.

## Siguiente paso

El usuario lanza esta ejecución en una sesión aparte (primer ODD de la serie H8, aislamiento de pruebas, H2). Empezar por la conciliación: `mem_context`, `mem_search` del topic, este fichero, `git fetch --prune origin`, y comprobar que las líneas citadas siguen siendo válidas. Después, T1 con permiso del usuario para el runner de Windows.
