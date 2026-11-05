# ODD: Absorción upstream — seguridad y corrupción de datos

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/seguridad-datos-upstream.md`.
> Espejo de recuperación en Engram: topic `odd/seguridad-datos-upstream/tasks`, proyecto `axiom`.
> Rama: `inc/seguridad-datos-upstream` (worktree `axiom-wt/seguridad-datos-upstream/axiom`).

## Objetivo

Portar desde gentle-ai (`origin/main` 3ea08cdf, posterior a v4.3.1) los arreglos de seguridad y de integridad de datos que corrigen fallos presentes hoy en Axiom (`main` f7158af0): reglas de denegación de secretos que no protegen el HOME, permisos de ficheros que se ensanchan al reescribir, el PATH de usuario de Windows que se aplana y números JSON que se corrompen al fusionar configuraciones.

## Problema y porqué

Axiom reescribe ficheros de configuración del usuario (settings de Claude, configuraciones MCP, `opencode.json`, el PATH del registro de Windows). Cada reescritura que pierde información (un modo 0600, una referencia `%USERPROFILE%`, un entero de 64 bits) o que protege la ruta equivocada (`.ssh/` del proyecto en vez de `~/.ssh/`) es un defecto de seguridad o de corrupción silenciosa que el usuario no detecta. Upstream ya los resolvió con pruebas; aquí se adaptan a las divergencias de Axiom.

## Alcance autorizado

- Paquetes: `internal/components/filemerge`, `internal/components/permissions`, `internal/components/uninstall`, `internal/system` y los llamadores directos de `filemerge.WriteFileAtomic` y de `unmarshalJSONObject` que se citan en cada tarea.
- Solo los PRs upstream de la tabla. Cualquier otro PR que aparezca como dependencia se anota como seguimiento, no se porta aquí.

## Restricciones

- Módulo `github.com/IGutierrezZ/axiom/v3`: reescribir todo import `/v4` de upstream.
- Marca propia: ningún identificador visible `GentleAI`/`gentle-ai` en nombres de mensajes, claves de registro de pruebas o textos.
- Sin telemetría.
- No se modifica `docs/upstream-absorption-ledger.md` (decisión D4): los PRs absorbidos se registran en este documento.
- No se compilan binarios sueltos (el antivirus borra `axiom.exe`): solo `go test` en proceso.
- Las pruebas de bits de modo POSIX se saltan en Windows (`runtime.GOOS == "windows"`), como en upstream; donde sea posible se añade una prueba portable equivalente (por ejemplo, `Changed=false` con contenido idéntico).

## Dependencia con P1 (resuelta)

P1 (`inc/distribucion-un-comando`) se fusionó en la PR #220 (release v3.6.0). La rama está rebasada sobre `origin/main` 7235e6b6 (2026-11-05). Las referencias quedan así:

- **T3:** el párrafo de rutas sensibles está en `docs/quickstart.md:101`. Ya se puede editar.
- **T2:** `internal/update/upgrade/strategy.go:776` llama a `system.AddToUserPath`, cuyo comportamiento cambia T2. T2 ejecuta también `go test ./internal/update/...`.
- T1, T4 y T5 nunca dependieron de P1. `WriteStreamAtomic` (`upgrade/download.go:230,394`) queda fuera de T5.

## Orden entre bloques (cruce de ficheros, 2026-11-05)

Posición de este bloque (SEG) frente a las demás sesiones de la absorción upstream y de la hoja de ruta P2–P6:

| Tarea | Cuándo | Motivo |
|---|---|---|
| T1 (números JSON) y T4 (deny en el overlay) | En paralelo, desde ya | No comparten ficheros con ningún otro bloque |
| T2 (PATH de Windows) | Pronto | La tarea T7 del bloque de arreglos (instaladores) depende de su criterio sobre `REG_EXPAND_SZ`. P2 (actualización desde la UI) pasa por `strategy.go:776` → `system.AddToUserPath`: T2 entra antes que P2, o ambas se prueban juntas |
| T3 (`permissions/inject.go`, `uninstall/service.go` `rewriteManagedPermissions`) | Antes de T5 | En `permissions/inject.go` coincide con la tarea T5 del bloque de adaptadores (Copilot CLI), que entra después de T3 |
| T5 (`WriteFileAtomic`, unos 12 llamadores) | Última del bloque y última de la cadena compartida | Va después de las tareas T3 y T4 del bloque de arreglos, que tocan `cli/sync.go` y `uninstall/service.go`. La tarea T5 del bloque de adaptadores hereda la semántica de modo de fichero de esta T5 |

Consecuencia sobre el orden interno: T1 y T4 pueden ir a la vez (mismo PR 1), T2 sube de prioridad y T5 espera además a esas dos tareas del bloque de arreglos.

## Veredictos verificados (2026-11-04)

| PR upstream | Merge | Fallo en Axiom (evidencia) | Veredicto | Tarea |
|---|---|---|---|---|
| #5240 números JSON exactos | `628813e0` | `internal/components/filemerge/json_merge.go:195-211` decodifica sin `UseNumber`; igual en `internal/components/uninstall/cleaners.go:232-246` y `internal/agents/pi/adapter.go:137,404` | Adoptar | T1 |
| #5625 PATH conserva `REG_EXPAND_SZ` | `e8642e14` | `internal/system/path.go:86,138,167` usa `[Environment]::SetEnvironmentVariable(..., 'User')`: escribe `REG_SZ` y valores expandidos | Adoptar | T2 |
| #5632 PATH compara entradas expandidas | `93d6f9f1` | mismas líneas: comparación literal | Adoptar | T2 |
| #5867 deny de secretos anclado a `~/` | `5522f77d` | `internal/components/permissions/inject.go:50-59`: `Read(.ssh/*)`, `.credentials/*`, `Library/Keychains/*`, `.aws/credentials`, `.config/gh/hosts.yml`, relativas al proyecto | Adaptar | T3 |
| #5311 no relajar un deny por defecto del usuario | `eb725304` | `internal/components/permissions/inject.go:63-78`: las reglas `ask` del overlay de OpenCode/Kilo se imponen a un `permission."*": "deny"` del usuario | Adaptar | T4 |
| #5014 `WriteFileAtomic` preserva el modo | `533c3872` | `internal/components/filemerge/writer.go:178` aplica `tmp.Chmod(perm)` siempre; un 0600 pasa a 0644 | Adaptar | T5 |
| #5024 (solo `enforceFileMode`) | `eb462d2a` | sin equivalente; el arreglo de TOML del mismo PR queda fuera | Adaptar en parte | T5 |
| #5629 conservar deny existentes de Claude | `ee972760` | No hay fallo: Axiom ya une las listas (`inject.go:196-199`, `filemerge.UnionJSONArrays`, `TestInjectClaudeCodeUnionsUserDenyList`) | Descartar: su regla de no sembrar en una lista existente contradice `docs/components.md:16` | — |

## Tareas

Orden de ejecución: **T1 → T4 → T2 → T3 → T5**, que sigue los cortes de PR de «Entrega». T4 va tras T1 porque reutiliza `UnmarshalJSONObject` ya con números exactos; T5 va el último por ser el de más llamadores. T1 y T5 tocan `filemerge` en ficheros distintos; T3 y T4 tocan `permissions/inject.go` en ramas distintas (Claude y OpenCode/Kilo).

- [x] **T1 · Números JSON exactos al fusionar (#5240)**
  - `filemerge`: nueva `DecodeStrictJSONObject` con `UseNumber`; `unmarshalJSONObject` la usa **conservando el respaldo `normalizeJSON` (JSONC) de Axiom**, que upstream no tiene; `rejectDuplicateJSONKeys` con `UseNumber` si existe en Axiom.
  - Llamadores: `uninstall/cleaners.go:232`, `agents/pi/adapter.go:137,404`.
  - Consumidores de `float64`: `engram/healthprobe.go` (`stdioTimeoutFromConfig`, ~`:71`) acepta `json.Number`. Auditar con `rg` cualquier otro *type switch* numérico tras decodificar settings.
  - Pruebas: `TestJSONRewritesPreserveExactNumberTokens` (`9007199254740993`, `1.10`, `1E+2`), `TestUnmarshalJSONObjectKeepsStrictSingleDocument`, un caso JSONC con números, `TestProvisionEngramMCPPreservesExactNumberTokens`, `TestPersistedJSONTimeoutIsRead`.
  - Ruta: delegada (escritor único; 4+ ficheros).
  - **Hecho (2026-11-05)**: 6 ficheros, +243/−12. `DecodeStrictJSONObject` con `UseNumber`; `unmarshalJSONObject` conserva el respaldo `normalizeJSON` y también decodifica con `UseNumber`. `readPiJSONObject` (`pi/adapter.go:~404`) migrado. `stdioTimeoutFromConfig` acepta `json.Number`.
  - Sin cambios donde ya era correcto o no aplica: `uninstall/cleaners.go` ya usaba `UseNumber`; `pi/adapter.go:137` solo localiza rutas; Axiom no tiene `rejectDuplicateJSONKeys` en `filemerge`.
  - Auditoría de *type switches* numéricos: solo había que adaptar `healthprobe.go:73`; `pi_codegraph.go:615` (JSON-RPC) y `dashboard_test.go:692` quedan fuera.
  - Pruebas: `TestJSONRewritesPreserveExactNumberTokens`, `TestUnmarshalJSONObjectKeepsStrictSingleDocument`, `TestUnmarshalJSONObjectJSONCKeepsExactNumbers` (propia de Axiom), `TestProvisionEngramMCPPreservesExactNumberTokens` (Pi: `settings.json` y `npm/package.json`), `TestPersistedJSONTimeoutIsRead`.
  - Evidencia: `go test` ok en uninstall, agents, engram, mcp, permissions y update; `go vet` y `gofmt -l` limpios. En `filemerge` fallan `TestWriteFileAtomicFollowsDirectoryJunction` y `TestWriteFileAtomicFollowsRelativeDirectoryJunction` (`createtemp ... file already exists`): **fallo ambiental previo**, reproducido por el orquestador en `main` 7235e6b6 sin este cambio. El resto de `filemerge` pasa. Comprobación del orquestador: `go test ./internal/components/filemerge/ -run "Number|StrictSingle|JSONC"` ok. `axiom review assess`: medium; con RDD apagado basta la verificación del escritor más esta comprobación.
- [x] **T2 · PATH de usuario en Windows sin aplanar (#5625, #5632)**
  - `internal/system/path.go`: envoltorio `persistentUserPathScript` que abre `HKCU\Environment`, lee con `DoNotExpandEnvironmentNames`, conserva `GetValueKind` (por defecto `ExpandString`), escribe con `SetValue` y emite `WM_SETTINGCHANGE` solo si hubo cambio; `Test-UserPathEntry` compara literal o expandido en añadir, quitar y priorizar (priorizar conserva la forma literal).
  - Conservar `skipPersistentUserPathWrite` y `NoPersistentPathEnvVar`. Nombre de la notificación con marca Axiom.
  - Pruebas en `internal/system/path_registry_test.go` con el *seam* `useWindowsUserPathSeam` existente y una clave temporal `HKCU\Software\Axiom-PathTest-<pid>-<ns>` borrada en `finally`; salto si `GOOS != "windows"` o `testing.Short()`; aserción de que ningún script toca `SubKey('Environment'`: `TestUserPathPreservesRegistryValue`, `TestUserPathMatchesExpandedEntries`, `TestPrioritizeUserPathPreservesLiteralSemantics`, `TestUserPathScriptInjectionRequiresAnchor`, `TestUserPathNotificationFailureRetainsRollbackOwnership`. Ajustar las pruebas de `path_test.go` que comparan el texto del script.
  - Seguimiento fuera de alcance: `UserPathEntries` (`path.go:182`) sigue leyendo valores expandidos, igual que upstream.
  - Ruta: inline (un fichero de código y sus pruebas) o delegada si el ajuste de `path_test.go` crece.
  - **Hecho (2026-11-05)**, ruta delegada (disparador de escritor: 3 ficheros). `path.go` +44/−20, `path_test.go` +5/−2, `path_registry_test.go` nuevo (332 líneas). Tipo C# `AxiomPathNotification`, clave de prueba `Software\Axiom-PathTest-<pid>-<ns>`.
  - Evidencia del escritor: `go test ./internal/system/...` ok (50 subtests de registro reales, 0 saltados); `go test -short ./internal/system/...` ok; `go test ./internal/update/...` ok; `go vet` limpio; sin claves `Axiom-PathTest` residuales en HKCU.
  - Comprobación del orquestador: `go test ./internal/system/ -run "TestUserPath|TestPrioritizeUserPath" -count=1` ok. `axiom review assess`: medium (`executable_change`); con RDD apagado basta la verificación del escritor más esta comprobación.
  - Nota: `rg GentleAI internal/system` solo encuentra `env.go:10` (`EnvChannelGentleAI`), anterior a esta tarea.
- [x] **T3 · Deny de secretos de Claude anclado al HOME (#5867)**
  - `permissions/inject.go`: reglas `Read/Edit(~/.ssh/**)`, `~/.credentials/**`, `~/Library/Keychains/**`, `~/.aws/credentials`, `~/.config/gh/hosts.yml`.
  - Migración: tabla `staleClaudeCodeDenyRules` y `anchorStaleClaudeCodeDenyRules`, aplicadas en `mergeJSONFile` (`inject.go:191`) antes de `filemerge.UnionJSONArrays`; reescribe en su sitio solo las coincidencias exactas, sin duplicar, sin tocar el resto ni su orden.
  - **Adaptación propia de Axiom:** `uninstall/service.go` (`rewriteManagedPermissions`) debe retirar también las reglas antiguas de instalaciones previas.
  - Documentación: alinear `docs/components.md:16` y el párrafo de rutas sensibles de `docs/quickstart.md:101` con la sintaxis real.
  - Cambio de comportamiento aceptado: un `.ssh/` dentro del proyecto deja de estar denegado.
  - Pruebas: `TestInjectClaudeCodeAnchorsStaleHomeDenyRules` (orden, sin duplicados, `Read(custom/**)` intacta, segunda inyección `Changed=false`); actualizar `TestInjectClaudeCodeSensitivePathsDenied` y `TestInjectClaudeCodeUnionsUserDenyList`; prueba de desinstalación que retire reglas antiguas y nuevas y conserve las del usuario.
  - Ruta: delegada (código, desinstalación y documentación).
  - **Hecho (2026-11-05)**: 6 ficheros, +208/−26.
  - Código: `staleClaudeCodeDenyRules` y `anchorStaleClaudeCodeDenyRules`, como en upstream. `withAnchoredClaudeCodeDeny` reescribe la lista del usuario solo como entrada de `UnionJSONArrays` (rama `union`, solo Claude Code). La fusión sustituye la lista entera, así que en el fichero final no quedan reglas antiguas.
  - Desinstalación: nueva `permissions.LegacyOverlay(agentID)` (solo Claude Code); `rewriteManagedPermissions` retira el overlay actual y después las reglas antiguas.
  - Documentación: `docs/components.md:16` y `docs/quickstart.md:101` describen las rutas reales por agente. Desaparecen las que Axiom no escribía (`~/.ssh/**/*`, `**/*.p12`, `**/*.pfx`).
  - Pruebas: `TestInjectClaudeCodeAnchorsStaleHomeDenyRules`, `TestLegacyOverlayOnlyForClaudeCode` y una prueba de desinstalación con reglas antiguas, nuevas y del usuario. Actualizadas `TestInjectClaudeCodeSensitivePathsDenied`, `TestInjectClaudeCodeUnionsUserDenyList` y otra comprobación de `Read(.ssh/*)`.
  - Evidencia: `go test` de permissions y uninstall ok (repetido por el orquestador); `go test ./internal/cli/ -run "Permission|Uninstall|Sync" -timeout 40m` ok en 136,7 s (ejecutado por el orquestador); `go vet` y `gofmt -l` limpios; `rg "Read\(\.ssh|Edit\(\.ssh"` solo encuentra la tabla de reglas antiguas y las semillas de las pruebas.
  - `go test ./internal/components/...`: además del fallo ambiental de uniones de `filemerge`, falló una vez `communitytool` `TestPiCodeGraphPendingProbePreservesConfiguredFiles` por timeout de la sonda MCP con la máquina saturada por otras sesiones; en solitario pasa con y sin el cambio. `axiom review assess`: medium.
- [x] **T4 · El overlay de OpenCode/Kilo no relaja el deny por defecto del usuario (#5311)**
  - `permissions/inject.go`: `withoutRulesLooserThanUserDefault(base, overlay)` lee `permission."*"` (cadena o mapa `{"*": ...}`), ordena `allow < ask < deny` y elimina del overlay los patrones menos estrictos; envuelve el merge de OpenCode/Kilo. Usa `filemerge.UnmarshalJSONObject` (`json_merge.go:216`), que ya devuelve números exactos tras T1.
  - Pruebas: `TestInjectKeepsUserDenyByDefault` (OpenCode y Kilocode), `TestInjectKeepsUserAskByDefault`, `TestInjectToleratesPatternMapRootDefault`. Comprobar antes que el helper `remoteAction` de upstream tiene equivalente en `inject_test.go`.
  - Ruta: inline (un fichero de código y su prueba). Se delegó igualmente para mantener limpio el contexto del orquestador.
  - **Hecho (2026-11-05)**: `inject.go` +44 y `inject_test.go` +119. El filtro se aplica en `mergeJSONFile`, dentro de la rama `defaults`, que solo usan OpenCode y Kilo Code; Claude queda igual. Pruebas: `TestInjectKeepsUserDenyByDefault` (OpenCode y Kilocode, con idempotencia), `TestInjectKeepsUserAskByDefault`, `TestInjectToleratesPatternMapRootDefault` y `TestInjectKeepsOverlayIntactUnderUserAllowByDefault`.
  - **Invariante:** como upstream, el overlay filtrado se vuelve a serializar con `json.Marshal`, que ordena las claves. Hoy es inocuo porque cada mapa del overlay tiene una sola acción (`bash` todo `ask`, `read` todo `deny`). Si un mapa llegara a mezclar acciones, el orden importaría, porque en OpenCode gana la última coincidencia.
  - **Riesgo previo, no agravado:** `removeOverlayEntries`, en la desinstalación, retira una regla del usuario que coincida exactamente en patrón y valor con una del overlay. Corregirlo exigiría registrar qué se escribió. Queda como seguimiento.
  - Evidencia: `go test ./internal/components/permissions/...` ok (repetido por el orquestador); `go test ./internal/components/uninstall/...` ok; `go test ./internal/cli/... -run "Permission|Uninstall"` ok; `go vet` y `gofmt -l` limpios. El paquete `internal/cli` completo agotó los 10 minutos con un pánico en `runReviewCommandContext` (revisión nativa), ajeno a esta tarea y no comprobado sobre `main`. `axiom review assess`: medium.
- [x] **T5 · Escritura atómica que preserva el modo (#5014 y `enforceFileMode` de #5024)**
  - `filemerge/writer.go`: `ExistingFileMode` (de upstream `209d8867`); `WriteFileAtomic` preserva el modo de un fichero existente y aplica `perm` solo al crear; nueva `WriteFileAtomicMode` que fuerza `perm` también con contenido idéntico (`perm == 0` → 0600); `enforceFileMode` informa `Changed=true` solo si el modo en disco cambia.
  - Reclasificar llamadores que deben **forzar** el modo y pasarlos a `WriteFileAtomicMode`: `backup/restore.go:440`, `mutationjournal/journal.go:46`, `cli/sync.go:1535`, `communitytool/pi_codegraph.go:32`, `gga/runtime.go:54,72,89`, `opencode/background.go:327` (comparar también el modo antes de saltar), `agents/claude/adapter.go:119` (eliminar el `os.Chmod` manual de `:122-129`), `uninstall/service.go:1571,1608`. El resto (configuraciones MCP, settings, `opencode.json`, limpiezas de desinstalación) pasa a preservar el modo, que es lo deseado.
  - `WriteStreamAtomic` no cambia.
  - Coordinación: el bloque de adaptadores (#5053, diferido) necesita `ExistingFileMode`; esta tarea lo deja disponible.
  - Pruebas (salto en Windows): `TestWriteFileAtomicPreservesExistingModeByDefault`, `…NewFileUsesRequestedPerm`, `…IdenticalContentNeverTouchesMode`, `TestWriteFileAtomicModeForcesRequestedPermOnExistingFile`, `…EnforcesPermOnIdenticalContent`, `…ZeroNeverWidens`, `…IdenticalContentAndModeIsNoOp`, `TestMergeJSONFilePreservesExistingModeOnRewrite`, las de lanzadores de `opencode/background.go` y la del 0600 de Claude. Prueba portable: contenido idéntico devuelve `Changed=false`. Reescribir los *fixtures* de `writer_test.go` que asumen 0644 forzado.
  - Ruta: delegada (unos 12 llamadores).
  - **Hecho (2026-11-05)**: 15 ficheros de código y pruebas, unas 500 líneas.
    - `writer.go`: `WriteFileAtomic` conserva el modo de un fichero existente; añade `WriteFileAtomicMode` (`perm == 0` → 0600), `ExistingFileMode` y `enforceFileMode`.
    - Migrados a `WriteFileAtomicMode`: `backup/restore.go`, `mutationjournal/journal.go` (escritura y `Restore`), `cli/sync.go` (`writeSyncFileAtomic`), `communitytool/pi_codegraph.go`, `gga/runtime.go` (3 scripts 0755), `opencode/background.go` (lanzadores), `agents/claude/adapter.go` (0600, sin el `os.Chmod` manual) y `uninstall/service.go` (2 restauraciones). Los llamadores con `0o644` no cambian y pasan a conservar el modo. `opencodeplugin/plugin.go` ya usaba `WriteFileAtomicMode`.
    - Adaptación propia de Axiom: en `opencode/background.go` el salto por contenido idéntico compara también el modo, salvo en un host Windows (`runtime.GOOS == "windows"`), para no reescribir el lanzador en cada ejecución.
    - `components/engram/inject.go` no se toca: `upsertCodexTableKeyBeforeMCPServers` sigue intacta.
  - Pruebas: las 7 de modo de `filemerge`, una portable (`TestWriteFileAtomicVariantsReportNoChangeOnIdenticalContent`), `TestMergeJSONFilePreservesExistingModeOnRewrite`, las dos del 0600 de Claude, gga, dos de lanzadores de opencode y una comprobación de modo en `backup/restore_test.go`.
  - Evidencia del escritor:
    - `go test` de `backup`, `components/...`, `agents`, `opencode`, `state` y `skillregistry` en verde, salvo fallos ambientales previos.
    - `go test ./internal/cli/ -run "Sync|Restore|Backup|Uninstall|Permission"` ok (130 s); `go vet ./internal/...` y `gofmt -l internal` limpios.
    - Fallos ambientales, que también ocurren en `main`: las dos pruebas de uniones de `filemerge` (mismo mensaje) y `backup` `TestSnapshotterSkipsUnixSockets`.
  - `axiom review assess`: **high** (`process_boundary`, 510 líneas). Con RDD apagado, verificación independiente de solo lectura: **PASS WITH ISSUES**, sin hallazgos altos ni medios; `writer.go` es equivalente a upstream y ningún llamador está mal clasificado.
    - El comentario obsoleto de `gga/runtime.go` se corrigió.
    - Quedan dos seguimientos de severidad baja: casi todas las pruebas de modo se saltan en Windows, así que el contraste «conservar vs forzar» solo lo cubre la CI de Linux/macOS (como en upstream); y `TestWriteFileAtomicModeZeroNeverWidens` acepta también 0000 y 0400.
  - Comprobación del orquestador: `go test ./internal/components/filemerge/ ./internal/agents/claude/ ./internal/opencode/` ok, salvo las dos uniones.
  - CI de la PR #232 (Linux): falló `TestInjectClaudeWritesUserConfigAndIsIdempotent` (`mcp`). La prueba afloja `~/.claude.json` a 0644 y esperaba `Changed=false` en la segunda inyección, pero con `enforceFileMode` la reparación solo de modo cuenta como cambio. Se porta el ajuste de upstream (#5014): la segunda ejecución informa `Changed` si se aflojó y una tercera debe ser un no-op. En Windows no se detectó porque el aflojado del modo se salta.

## Criterios de aceptación

- Cada fallo de la tabla tiene una prueba que falla sobre `main` f7158af0 y pasa tras su tarea (en Windows, las de modo POSIX constan como saltadas).
- `go test ./internal/components/filemerge/... ./internal/components/permissions/... ./internal/components/uninstall/... ./internal/system/... ./internal/agents/... ./internal/components/engram/... ./internal/update/...` en verde.
- `go vet` de los mismos paquetes en verde.
- `rg -n "gentle-ai/v4|GentleAI" internal/` sin coincidencias nuevas.
- Ninguna escritura en `docs/upstream-absorption-ledger.md`.

## Comprobaciones aplicables

- TDD: **desactivado** (fuente: `openspec/config.yaml:16`, `strict_tdd: false`). Ejecutor: `go test` en proceso. Se ejecutan las comprobaciones funcionales de cada tarea; las pruebas portadas se escriben junto al cambio.
- Revisión nativa (RDD): según el interruptor del usuario (`axiom review mode status`); si está encendido, se evalúa cada commit de unidad de trabajo con `axiom review assess --committed-only`.

## Entrega

- Previsión: unas 1.100 líneas autoradas (T1 ~230, T2 ~370, T3 ~130, T4 ~185, T5 ~250-300), la mayoría pruebas. Supera el presupuesto de ~400 líneas por PR.
- Estrategia: `ask-on-risk`. Antes del primer commit que supere el presupuesto se preguntará la estrategia de cadena (`stacked-to-main` o `feature-branch-chain`). Propuesta de cortes: PR 1 = T1+T4 (~415), PR 2 = T2 (~370), PR 3 = T3+T5 (~400-430).
- Un commit convencional por tarea, como mínimo, con pruebas y documentación en el mismo commit.

## Progreso

- 2026-11-04: alcance formalizado; veredictos verificados en ambos repos. Pendiente de validación del usuario antes de implementar.
- 2026-11-04: comprobado el solape con P1 (`inc/distribucion-un-comando`): T2 y la parte de documentación de T3 esperan a su fusión.
- 2026-11-04: **alcance y orden validados por el usuario** (T1 → T4 → T2 → T3 → T5).
- 2026-11-05: P1 fusionado (PR #220); rama rebasada sobre `origin/main` 7235e6b6; añadido el orden entre bloques. T5 espera además a las tareas T3 y T4 del bloque de arreglos. No se implementa nada hasta que lo autorice el usuario.
- 2026-11-05: el usuario autoriza arrancar las tareas desbloqueadas («me gustaria que arranquen las que pueden empezar»). Orden de ejecución ajustado: **T2 → T1 → T4 → T3**, con T2 primero porque P2 (actualización desde la UI) ya está en marcha y pasa por `strategy.go:776`. T5 no empieza hasta que se fusionen las tareas T3 y T4 del bloque de arreglos. Commits locales por tarea; push y PR los decide el usuario. La estrategia de cadena de PRs se decidirá al abrir el primer PR; un commit por tarea permite cualquiera de los cortes propuestos.

- 2026-11-05: hechas T2 (`ddcea220`), T1 (`dc047508`), T4 (`7e74d2aa`) y T3 (`2835415e`), todas en local y sin push. Acumulado: unas 1.040 líneas autoradas en 4 commits de código. T5 queda en espera.

- 2026-11-05: por decisión del usuario, una sola PR con `size:exception` (la misma preferencia que en P2). Rama rebasada sobre `origin/main` 8bf0fb8f; pruebas repetidas tras el rebase, todas en verde salvo el fallo ambiental de uniones de `filemerge`. **PR #228** con T1–T4. T5 irá en otra PR.

- 2026-11-05: PR #228 fusionada (merge `fbf96dcb`). Las tareas T3 y T4 del bloque de arreglos se fusionaron en las PR #229 y #231 (merge `3ad77567`). T5 queda desbloqueada; la arranca la misma autorización del usuario. Nueva rama `inc/seguridad-datos-upstream` desde `origin/main` 3ad77567.

- 2026-11-05: T5 con commit `579b9224`; el usuario autoriza el push y la PR: **PR #232** (`size:exception`, `type:bug`).

Plan stop: approved 2026-11-04; implementation continues

## Siguiente paso

T5 en esta rama. Aviso del bloque de arreglos: su T4 tocó `cli/sync.go`, `cli/run.go`, `components/sdd/inject.go` y `components/engram/inject.go`; esta última incluye `upstreamCodexTableKeyBeforeMCPServers` (`d290c8b0`), que no debe perderse.
