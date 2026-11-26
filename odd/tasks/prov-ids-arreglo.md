# prov-ids-arreglo

Tipo: bug

## Objetivo

Corregir los dos fallos del manejo de identificadores que aparecieron al archivar C-03-c (adopcion-ids), antes de la migración de las specs de Axiom a identificadores funcionales (C-03-e):

1. El escáner de provisionales de la integración toma por provisionales reales los ejemplos literales de `tasks.md` y de las deltas, y bloquea el archivado.
2. La comprobación de colisiones de los ADDED con identificador definitivo compara por cadena y, con patrones con `<m>`, no ve que `RF-1` y `RF-001` son el mismo identificador.

## Problema y porqué

Fallo 1. `PlanIDs` (`internal/livespec/ids.go`, paso b) recorre con `reqid.ScanProvisional` el texto entero de cada delta; `changeArtifacts` (`internal/livespec/change.go`) toma como cita cualquier provisional de cualquier línea de los ficheros de tareas, y `docCitations` (`internal/livespec/odd_ids.go`) cualquier línea del documento ODD fuera de sus vallas. `Compose` (`internal/openspecdoc/compose.go`) rechaza con `ProvisionalIDError` cualquier token en el cuerpo de un requisito. Un ejemplo literal (en prosa, en código en línea o en un bloque de código) acaba como `provisional_malformed` o `provisional_undeclared`. En C-03-c hubo que reescribir los ejemplos con ordinales simbólicos (`prov:<serie>:<k>`) para poder archivar (Engram #2631 y #2638).

Fallo 2. `checkDefinitiveAdded` (`internal/livespec/ids.go:401`) busca el definitivo con `Registry.Lookup` (por cadena) y marca los repetidos del propio cambio con mapas por cadena. Con un patrón con `<m>` (ancho libre), un ADDED con `RF-1` no colisiona con un `RF-001` ya integrado aunque sean el mismo identificador según la decisión (c′) de C-03-c (duplicados por serie y número). `Registry.LookupIdentity` ya existe y no se usaba aquí.

## Alcance

- Escáner: cuentan solo los provisionales en posición de identificador, que son las referencias que el diseño de C-03-b reescribe (REQ-47.4):
  - la delta: títulos, cuerpos y preámbulo **fuera de código** (vallas y código en línea);
  - las líneas `Cubre:` de los ficheros de tareas y del documento ODD, fuera de vallas;
  - la columna `Requisito` de la matriz de `verify-report*.md` (sin cambios);
  - la valla `openspec` de `## Spec viva` del documento ODD, fuera de código en línea.
  El resto de los ficheros de tareas y del documento ODD ya no se valida, como `proposal.md` o `design.md`.
- `Compose` aplica la misma regla: un token dentro de código del cuerpo de un requisito es un ejemplo, no un provisional.
- Colisiones: identidad por serie y número en `checkDefinitiveAdded` (registro, repetidos del propio cambio, traza y `nextFree`).

Fuera de alcance:

- `IDMap.Substitute` no cambia: un token que coincide con un provisional declarado se sustituye también dentro de código (es una mención del mismo provisional; un ejemplo que no declara ningún ADDED no coincide con ninguna clave y queda intacto).
- Los bloques de código en línea que abarcan varias líneas: el escáner detecta el código en línea dentro de cada línea.

## Restricciones

- Cortex XDR borra ejecutables Go sueltos: sin `go build -o`, `go run` ni `go install`.
- Pruebas localizadas: `./internal/reqid`, `./internal/livespec`, `./internal/openspecdoc` y, en `./internal/cli`, solo con `-run`. Nunca la suite entera en local.
- El documento no contiene ningún provisional literal: el `axiom` del `PATH` (main, sin este arreglo) lo integra al cerrar.

## Decisiones

- **Qué es código.** Un bloque vallado (`` ``` `` o `~~~`, con las mismas reglas de apertura y cierre que `openspecdoc`) o un tramo de código en línea (una tirada de comillas invertidas cerrada por otra de la misma longitud en la misma línea). Una tirada sin cierre es literal.
- **Nueva función `reqid.ScanProvisionalRefs`.** `ScanProvisional` se conserva para los nombres (títulos) y para `isoevidence`; la nueva salta el código y la usan el plan (delta), `Compose` y la valla de la spec viva ODD.
- **Tareas y documento ODD.** Las citas salen de `isoevidence.ParseCubre`, la misma lectura de líneas `Cubre:` que usa `RewriteCubre` para reescribirlas.
- **Colisión (T2).** `checkDefinitiveAdded` usa `Registry.LookupIdentity` y claves `identityKey` en los mapas de repetidos, de traza y de ocupados; `nextFree` salta los números ocupados por identidad.

## TDD

Modo: desactivado (sin configuración en `axiom.yaml` ni petición). Cada tarea escribe primero la prueba que reproduce el fallo y se observa en rojo antes del arreglo. Ejecutor: `go test` por paquete, `-count=1`.

## Tareas

- [x] T1. Escáner: solo provisionales en posición de identificador y fuera de código. Ruta: inline (cambio localizado en `reqid`, `livespec` y `openspecdoc`).
  - Pruebas: `TestScanProvisionalRefsSkipsCode` (`reqid`: prosa, código en línea con tiradas simples y dobles, tirada sin cierre, vallas de comillas y de virgulillas, valla sin cerrar, cierre más corto); `TestPlanChangeIgnoresProvisionalExamplesInCodeAndTaskProse` (SDD: ejemplos en código en el cuerpo de un ADDED y en prosa y código de `tasks.md`; la spec viva conserva el ejemplo en código), `TestPlanChangeStillRejectsAnUndeclaredProvisionalInDeltaProse` (la prosa de la delta sigue validándose); `TestODDDocIgnoresProvisionalExamplesOutsideIdentifierPositions` y `TestODDDocRewriteRejectsAnUndeclaredProvisionalInACubreLine` (sustituye a la prueba que fijaba la validación del resto del documento). RED observado: `provisional_malformed` de la clave vacía en el ejemplo, en SDD y en ODD; `reqid.ScanProvisionalRefs` no existía.
  - Arreglo: `reqid.ScanProvisionalRefs` (salta vallas y código en línea); el paso b de `PlanIDs` y el guardián de `Compose` la usan; las citas de tareas y del documento ODD salen de `cubreCitations` (`isoevidence.ParseCubre`).
  - Evidencia: `go test -count=1 ./internal/reqid ./internal/livespec ./internal/openspecdoc ./internal/isoevidence` → `ok`; `go test -count=1 ./internal/cli -run 'ArchiveIntegrate|ArchiveCheck|ArchiveCompose|Provisional|SDDArchive|ODDIntegrat'` → `ok` (119 s); `gofmt -l` y `go vet` limpios.
- [x] T2. Colisión por identidad (serie y número) en `checkDefinitiveAdded`. Ruta: inline.
  - Pruebas: `TestPlanIDsDefinitiveAddedCollisionsCompareByIdentityWithFreeWidth` (`RF-<m>`: un ADDED con `RF-1` frente a `RF-001` de la canónica colisiona y el siguiente libre salta el `RF-02` del propio cambio, `RF-3`; `RF-5` y `RF-05` en dos capacidades del cambio colisionan) y el subcaso «repetido en otra capacidad con otro ancho» de `TestCheckArchivedTraceReportsADefinitiveThatAppearsMoreThanOnce` (`REQ-30.3` de la traza frente a `REQ-30.03` ajeno). RED observado: el plan aceptaba `RF-1` sin colisión y la comprobación de la traza archivada no veía el duplicado.
  - Arreglo: `checkDefinitiveAdded` compara con `Registry.LookupIdentity` y claves `identityKey` (traza, ocupados y repetidos); `nextFree` salta los ocupados por identidad; `resolveTraced` y `CheckArchivedTrace`, el mismo defecto, usan también `LookupIdentity`. `Registry.Lookup` queda sin uso en producción y pasa a `export_test.go` (la CI tiene un *ratchet* de código muerto).
  - Evidencia: `go test -count=1 ./internal/livespec ./internal/adoption/...` → `ok`; `go test -count=1 ./internal/cli -run 'ArchiveIntegrate|ArchiveCheck|ArchiveCompose|ArchiveAdopt|Provisional|SDDArchive|ODDIntegrat'` → `ok` (96 s); `go vet` de `livespec`, `cli`, `adoption` y `livingdoc` limpio.
- [ ] T3. Cierre: spec viva, `axiom archive integrate --odd prov-ids-arreglo`, PR.

## Criterios de aceptación

- Prueba que reproduce cada fallo (rojo antes, verde después).
- `go test -count=1` en verde para `./internal/reqid`, `./internal/livespec`, `./internal/openspecdoc` y `./internal/cli` con `-run` sobre archive.
- Un commit por tarea.

## Progreso

- Documento creado; T1 en curso.
