# Tareas: arranque de un cambio desde un work item existente (C-07-d)

> **Cambio:** `arranque-desde-wi`. **Rol:** único `fullstack` (kickoff `continuous`). **Idioma:** castellano de España; identificadores, comandos, claves y rutas en inglés.
> Fuentes: `proposal.md`, `specs/*/spec.md`, `design.md` (D-01..D-29, matrices y «Huecos de spec»).
> **Modo TDD:** no estricto (`openspec/config.yaml` declara `strict_tdd: false`; Engram `sdd-init/axiom` no registra `strict_tdd: true`). Aun así, cuando es natural, la prueba RED se escribe y se observa fallar antes del código de producción; las pruebas de la matriz de amenazas aplicables son RED-primero obligatorio.
> **Reglas comunes a todas las tareas:** un commit Conventional Commit por tarea, sin atribución de IA; pruebas siempre localizadas con `go test <paquete> -run '<patrón>'` (en `internal/cli` nunca el paquete entero); nada de compilar binarios sueltos (Cortex); ninguna prueba contra Azure DevOps real (todo HTTP va a `adotest` con `localOnlyTransport`); la batería completa se deja a la CI. Cada tarea marca sus casillas solo tras observar el resultado.

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | XL |
| Estimated changed lines | 9.000 - 13.000 (adiciones + borrados, sin ficheros generados; los goldens se excluyen) |

### Work Units

| Unit | Goal | Commit | Focused test command | Runtime harness | Rollback boundary | Ruta |
|------|------|--------|----------------------|-----------------|-------------------|------|
| T0 | Alinear las deltas de spec con los huecos 1-9 del diseño | `docs(spec): alinear las deltas de arranque-desde-wi con los huecos del diseño` | Relectura estructural: `Grep` de `prov:74:` (27 IDs únicos, sin cambios) | N/A: solo documentación | `specs/*/spec.md` y nota de huecos del `design.md` | inline |
| T1 | Manifiesto, `item_keys`, `RefAlias`, capacidades | `feat(integrations): declarar item_keys, ref_alias y capacidades tags y attachments en azure-devops` | `go test ./internal/integrations/manifest/... ./internal/workspace/... -run 'TestParseTypePrefixes\|TestAzureDevOpsManifest\|TestDefaultRegistryMatchesCatalog\|TestItemKeys'` | N/A: sin comportamiento de ejecución observable; prueba de carga de `axiom.yaml` en `workspace` | Cambios en `manifest/` y `workspace/integrations.go` | inline |
| T2 | Puerto `WorkItemSource` + adaptador ADO + `adotest` | `feat(integrations): puerto WorkItemSource con lectura tipada, descarga binaria y enlace de vuelta` | `go test ./internal/integrations/... -run 'TestWorkItemPort\|TestLegacyPluginStillRegisters\|TestParseItemRef\|TestReadItem\|TestStateCategory\|TestDownload\|TestLinkBack\|TestAdotest\|TestLinkWorkItem'` | `adotest` en proceso con `localOnlyTransport` (peticiones al servidor local) | `integrations/workitems.go`, `links.go`, `azuredevops/*` | inline (candidata a delegada por contexto si crece) |
| T3 | `htmlmd`, neutralización `AXIOM::`, render | `feat(changestart): convertir HTML a Markdown y marcar el contenido importado como dato no fiable` | `go test ./internal/changestart/... -run 'TestConvert\|TestNeutralize\|TestRender'` y `go test ./internal/changestart/htmlmd -run '^$' -fuzz FuzzConvert -fuzztime 20s` | Fuzz acotado en proceso | `internal/changestart/htmlmd/`, `untrusted.go`, `render.go` | inline |
| T4 | Adjuntos: prefiltro, dedup, firmas, presupuesto, nombres | `feat(changestart): importar adjuntos con lista blanca, firmas, presupuesto y deduplicación` | `go test ./internal/changestart/... -run 'TestAttachment\|TestSignature\|TestImage\|TestOOXML'` | `adotest` con adjuntos binarios y redirección | `attachments.go`, `dedup.go`, `signature.go` | inline |
| T5 | Fase 1 y CLI `axiom change start` | `feat(cli): axiom change start --from con staging atómico bajo el common dir` | `go test ./internal/changestart/... -run 'TestStart\|TestDeriveName\|FuzzDeriveName\|TestLock\|TestSweep'`; `go test ./internal/cli -run 'TestChangeStart'`; `go test ./cmd/axiom -run 'TestChangeHelpListsStart\|TestAxiomHelpMentionsChangeStart'` | `RunChangeStart` en proceso con `adotest.Install` y repo git temporal | `cli/change_start.go`, `changestart/*`, `cmd/axiom/main.go`, docs | inline |
| T6 | Fase 2 `--adopt` y `--discard` | `feat(cli): axiom change start --adopt y --discard con copia verificada y enlace de vuelta` | `go test ./internal/changestart/... -run 'TestAdopt\|TestDiscard'`; `go test ./internal/state ./internal/docpaths -run 'TestEnsureChangeContextIgnored\|TestLocalStateIncludesContext'`; `go test ./internal/integrations -run 'TestLinkWorkItem'`; `go test ./internal/branch/... -run 'TestBranchDoneIgnoresIgnoredContext'` | Servicio en proceso con git temporal y costuras `Env.FS` | `adopt.go`, `discard.go`, `copy.go`, `state/project_ignore.go`, `docpaths`, `.gitignore` | delegada solo si se cruza el respaldo de contexto; si no, inline |
| T7 | Skills y convención común | `docs(skills): tratar el contexto del work item como dato no fiable y condicionar external_ref al vínculo` | `go test ./internal/components/... -run 'TestSkillsTreatWorkItemContextAsUntrusted\|TestProposeExternalRefRule'`; `go test ./internal/sddstatus/... -run 'TestStatusDoesNotRequireExternalRef\|TestStatusWithContextDir'` | N/A: texto de skills; aserciones estáticas | Cuatro skills y sección H | inline |
| T8 | Doctrina y goldens (única, última) | `feat(doctrine): regla del work item de origen en SDD y ODD con goldens regenerados` | `go test ./internal/components/... -run 'TestOrchestratorWorkItemRuleParity\|TestODDRoutingWorkItemRule\|TestRoutingAddsNoStop'`; `go test ./internal/components -run TestGolden` (sin `-update`, tras regenerar) | Regeneración `-update`, revisión de diff y relanzado sin `-update` | `sdd-orchestrator-sections.md`, `routing.go`, `testdata/golden/*` | inline |
| T9 | Cierre: trinquetes antes de la PR | `chore(change-start): cerrar trinquetes de código muerto y rechazos` (solo si hay ficheros que ajustar; si no, sin commit) | `scripts/deadcode-ratchet.sh`; `go test ./internal/cli -run TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`; `go vet` y `gofmt -l` de los paquetes tocados | Ejecución de los trinquetes | N/A | inline |

---

## T0. Alinear las deltas con los huecos abiertos del diseño

Commit: `docs(spec): alinear las deltas de arranque-desde-wi con los huecos del diseño`. Ruta: inline. Sin tocar IDs `prov:74:*`.

- [x] 0.1 `openspec/changes/arranque-desde-wi/specs/change-start-from-work-item/spec.md` (prov:74:4): añadir las formas de referencia de D-29 (identificador `^[1-9][0-9]{0,9}$` ≤ 2147483647; URL `_workitems/edit`, `_apis/wit/workitems/{id}` y cualquier ruta bajo `/{org}/` con `workitem={id}`; fragmento y barra final ignorados; proyecto de la URL igual al configurado; `{org}.visualstudio.com` no aceptado) con escenarios de tablero, otro proyecto y `visualstudio.com`. Hueco 1.
- [x] 0.2 Mismo fichero (prov:74:14): permitir ejecutar el paso 4 (`info/exclude`) antes del paso 2 sin cambiar el orden observable del resto, o declarar explícitamente la ventana por muerte del proceso; sustituir «renombrar y caer a copia» por «copiar y verificar siempre; el staging queda intacto hasta el paso 6». Huecos 2 y 3.
- [x] 0.3 Mismo fichero (prov:74:13/14, `already_adopted`): definir `already_adopted` también en la fase 1 (cambio del mismo nombre con el WI ya adoptado, sin descarga ni escrituras, aviso con `--adopt` si `backlink` no es `done`). Hueco 4.
- [x] 0.4 Mismo fichero (prov:74:14, paso 6): fallo al retirar el staging = aviso `staging_not_retired` con salida 0 y reintento de `--adopt`. Hueco 5.
- [x] 0.5 Mismo fichero (prov:74:17): añadir el cerrojo por identidad del WI (`item-<plugin>-<id>`) junto al del cambio, con orden fijo WI → cambio → integraciones. Hueco 6.
- [x] 0.6 Mismo fichero (prov:74:7): una etiqueta `axiom:<otro>` sin rastro local cuenta como abierta; el mensaje dice que el cambio no consta en este equipo y la continuación ofrece retirar la etiqueta o actualizar el checkout principal. Hueco 7.
- [x] 0.7 Mismo fichero (prov:74:9): perfil `bug` solo cuando el tipo mapeado es literalmente `Bug` (sin distinguir mayúsculas ni espacios extremos); nota de que otros nombres de tipo de defecto dan `incremento`. Hueco 8.
- [x] 0.8 Mismo fichero (prov:74:1): `git_repository_required` para `--from` no vacío, `--adopt` y `--discard` fuera de un repositorio git; la consulta sin argumentos y `--from` vacío no lo exigen. Hueco 9.
- [x] 0.9 `openspec/changes/arranque-desde-wi/design.md`: marcar en «Huecos de spec» los nueve huecos como «recogidos en la spec» sin alterar las decisiones D-xx.
- [x] 0.10 Verificación: `Grep` de `prov:74:` en `specs/` devuelve los mismos 27 identificadores sin duplicados ni pérdidas; relectura estructural de cada escenario nuevo (WHEN/THEN) y commit.

Cubre: prov:74:1, prov:74:4, prov:74:7, prov:74:9, prov:74:13, prov:74:14, prov:74:17 (alineación, sin IDs nuevos).

---

## T1. Manifiesto y configuración

Commit: `feat(integrations): declarar item_keys, ref_alias y capacidades tags y attachments en azure-devops`. Ruta: inline. Dependencias: ninguna (T0 no bloquea el código).

- [x] 1.1 RED: crear `internal/integrations/manifest/format_test.go` con `TestParseTypePrefixes` (tabla: elemento válido, sin `=`, dos `=`, tipo vacío, prefijo fuera de `^[A-Za-z0-9]{1,10}$`, tipo repetido, prefijo repetido sin distinguir mayúsculas) y observarla fallar.
- [x] 1.2 Crear `internal/integrations/manifest/format.go` con `ValueFormat`, `FormatTypePrefixPairs`, `TypePrefix`, `FormatProblem` y `ParseTypePrefixes` (D-07); GREEN de 1.1.
- [x] 1.3 Modificar `internal/integrations/manifest/manifest.go`: `ConfigField.Format`, `Manifest.RefAlias` (`ref_alias`, patrón `^[a-z][a-z0-9]{1,15}$`) y validación en `Manifest.Validate` (el `Default` cumple su formato). Pruebas en `internal/integrations/manifest/manifest_test.go`: `TestManifestRefAliasPattern`, `TestManifestDefaultMustMatchFormat`.
- [x] 1.4 Modificar `internal/integrations/manifest/azuredevops.go`: capacidades exactamente `items.read`, `discovery`, `tags`, `attachments`; `RefAlias: "ado"`; `item_keys` como `FieldStringList` con `Format: type_prefix_pairs` y `Default: ["User Story=US", "Bug=BUG"]`. Pruebas: `TestAzureDevOpsManifestCapabilities`, `TestAzureDevOpsManifestRefAlias`, `TestAzureDevOpsManifestItemKeys`, `TestDefaultRegistryMatchesCatalog`.
- [x] 1.5 Modificar `internal/workspace/integrations.go`: `stringList` aplica `Format` y convierte cada `FormatProblem` en error con clave, elemento y línea; aplicar el `Default` si falta `item_keys`. Pruebas en `internal/workspace/integrations_test.go`: `TestItemKeysValid`, `TestItemKeysDefaultApplied`, `TestItemKeysMalformed`, `TestItemKeysDuplicate`, `TestItemKeysPrefixInvalid`; comprobar que las pruebas existentes de `base_url`, `pat` y PAT por llamada siguen verdes.
- [x] 1.6 Ejecutar `go test ./internal/integrations/manifest/... ./internal/workspace/... -run 'TestParseTypePrefixes|TestManifest|TestAzureDevOpsManifest|TestDefaultRegistryMatchesCatalog|TestItemKeys'`, `go vet` y `gofmt -l` de ambos paquetes; commit.

Cubre: RF-ADO-01 (MOD), RF-ADO-02 (MOD), prov:74:9 (base del perfil).

---

## T2. Puerto `WorkItemSource`, adaptador ADO y `adotest`

Commit: `feat(integrations): puerto WorkItemSource con lectura tipada, descarga binaria y enlace de vuelta`. Ruta: inline. Dependencias: T1 (`item_keys` y alias).

- [ ] 2.1 RED: crear `internal/integrations/workitems_test.go` con `TestWorkItemPortDispatch` (cinco operaciones), `TestWorkItemPortUnsupported` (`port_unsupported` sin red), `TestWorkItemPortPanicRecovered`, `TestWorkItemPortSanitizesSecret`, `TestLegacyPluginStillRegisters`; observarlas fallar.
- [ ] 2.2 Crear `internal/integrations/workitems.go`: `PortWorkItems`, tipos neutrales (`ItemRelation`, `ItemAttachment`, `ExternalItem`, `ItemRequest`, `StateRequest`, `StateCategory`, `AttachmentRequest`, `Attachment`, `BackLinkRequest`, `BackLink`), interfaz `WorkItemSource`, centinelas (`ErrItemNotFound`, `ErrItemConflict`, `ErrResourceURLRejected`, `ErrAttachmentTooLarge`), `ResourceURLRejection` y las cinco funciones de despacho por `invokePort` (D-02, D-03). Modificar `internal/integrations/ports.go` (`PortWorkItems` y su etiqueta en `portLabel`). GREEN de 2.1.
- [ ] 2.3 Modificar `internal/integrations/azuredevops/adotest/workitems.go` y crear `internal/integrations/azuredevops/adotest/states.go` y `attachments.go`: `SeedStates`, `SeedAttachment` (sin `Content-Length`, sin `Content-Type`, redirección, estado), PATCH de `System.Tags` sin sembrar y `add /relations/-`, contadores de lecturas por ruta. Pruebas `TestAdotestRelationsTagsAttachments`, `TestAdotestAttachmentNoContentLength`, `TestAdotestAttachmentNoContentType`, `TestAdotestRedirect`, `TestAdotestAddRelations`, `TestAdotestStatesFailure` en `internal/integrations/azuredevops/adotest/adotest_test.go`.
- [ ] 2.4 RED: crear `internal/integrations/azuredevops/ref_test.go` con `TestParseItemRef` (tabla de D-29: id, `0`, cero inicial, > int32, `abc`; URL `_workitems/edit` con y sin proyecto, barra final, fragmento; `_apis/wit/workitems`; tablero `?workitem=`; otra organización, otro proyecto, otro host, `visualstudio.com`, `http`, `userinfo`). Crear `internal/integrations/azuredevops/ref.go` (D-29). GREEN.
- [ ] 2.5 Modificar `internal/integrations/azuredevops/client.go` (`request.absoluteURL`, `bodyLimit`, `binary`; `attempt` con `io.LimitReader(body, bodyLimit+1)` y `*TooLargeError`; `classify` en modo binario con `Content-Type` normalizado por `mime.ParseMediaType`, ausente = `application/octet-stream`; 3xx sigue siendo `AuthError`) y `internal/integrations/azuredevops/errors.go` (`ForeignURLError` con motivo cerrado, `TooLargeError`, conflicto persistente con `integrations.ErrItemConflict`; `NotFoundError.Is` sin cambios).
- [ ] 2.6 RED: crear `internal/integrations/azuredevops/download_test.go` con `TestDownloadOK`, `TestDownloadLimitAborts`, `TestDownloadForeignHostNoRequest`, `TestDownloadRejectionReasons` (cinco motivos, sin red y sin resolver el PAT), `TestDownloadMissingContentType`, `TestDownloadRedirectNotFollowed`, `TestDownloadErrorSanitized`. Crear `internal/integrations/azuredevops/download.go` con `validateResourceURL` y `Client.Download` (D-05). GREEN.
- [ ] 2.7 RED: crear `internal/integrations/azuredevops/workitem_port_test.go` con `TestReadItemFullTranslation`, `TestStateCategoryFromStates`, `TestStateCategoryUsesConfiguredTypeSpelling`, `TestStateCategoryUnknownState`, `TestReadItemNotFoundOnlyForItem`, `TestReadItemUnknownRelationAndMissingFields`, `TestReadItemForeignRelationNotRequested`, `TestReadItemRedactsPAT`. Modificar `internal/integrations/azuredevops/workitems.go` (`readItem` con `$expand=relations`, un único `workitemsbatch` tope 50, URL del WI desde la configuración, `stateCategory` con `url.PathEscape`) y crear `internal/integrations/azuredevops/workitem_port.go` (implementación del puerto y traducción de errores en la frontera, D-03, D-04). GREEN.
- [ ] 2.8 RED: crear `internal/integrations/azuredevops/linkback_test.go` con `TestLinkBackAddsKeepingOthers`, `TestLinkBackPresentNoWrite`, `TestLinkBackRevConflictRetries`, `TestLinkBackPersistentConflict`, `TestLinkBackOnlyTagsField`, `TestLinkBackAllowRefuses`. Crear `internal/integrations/azuredevops/linkback.go` (hasta 3 intentos con `test /rev`, `present` sin escritura, `Allow` por callback, D-06). GREEN.
- [ ] 2.9 RED: en `internal/integrations/links_test.go` añadir `TestLinkWorkItemCreatesFile`, `TestLinkWorkItemIdempotent`, `TestLinkWorkItemKeepsDeliveryLinks`, `TestLinkWorkItemRejectsUserinfo`. Modificar `internal/integrations/links.go` con `LinkWorkItem(f Folder, pluginID string, l Link, now time.Time) (bool, error)` (toma el cerrojo y reutiliza `mergeLinks` con la clave `change.start`) y `WorkItemLinks(f Folder) ([]Link, error)`. Las pruebas existentes de `mergeLinks` siguen verdes.
- [ ] 2.10 Ejecutar `go test ./internal/integrations/... -run 'TestWorkItemPort|TestLegacyPluginStillRegisters|TestParseItemRef|TestReadItem|TestStateCategory|TestDownload|TestLinkBack|TestAdotest|TestLinkWorkItem'`, `go vet` y `gofmt -l`; commit.

Cubre: prov:74:4 (resolución), prov:74:13 (PAT restringido en la descarga), prov:74:20, prov:74:21, prov:74:22, prov:74:23, RF-INT-09 (MOD), RF-ADO-16 (MOD), prov:74:19 (hermeticidad del adaptador).

---

## T3. `htmlmd`, neutralización `AXIOM::` y render del contexto

Commit: `feat(changestart): convertir HTML a Markdown y marcar el contenido importado como dato no fiable`. Ruta: inline. Dependencias: T2 (tipos neutrales para `render`).

- [ ] 3.1 RED: crear `internal/changestart/htmlmd/convert_test.go` con `TestConvertBasic`, `TestConvertHostile` (`script`, `style`, `iframe`, sin cierre, `<!--`), `TestConvertLinkSchemes` (solo `http`, `https`, `mailto`, sin `userinfo`), `TestConvertTables` (GFM y degradación con `colspan`), `TestConvertCodeFences`, `TestConvertEntities`, `TestConvertInputCap` (1 MiB) y `FuzzConvert` (sin pánico y sin etiqueta cruda).
- [ ] 3.2 Crear `internal/changestart/htmlmd/tokenize.go` y `internal/changestart/htmlmd/convert.go` (D-09): tokenizador conservador, subconjunto de etiquetas, `html.UnescapeString`, `<` seguido de letra, `/`, `!` o `?` escrito `&lt;`, lista ordenada de imágenes (`src`, `alt`). GREEN.
- [ ] 3.3 RED: crear `internal/changestart/untrusted_test.go` con `TestNeutralizeInjectedDelimiter`, `TestNeutralizeForgedHeader` (dos puntos de anchura completa `：`, `﹕`, `꞉`, `∶`, espacios intermedios, sin distinguir mayúsculas), `TestNeutralizeTitlesNamesReasons`, `TestNeutralizeZeroWidthAndBidi`, `TestInlineNeutral` (saltos de línea, tope de 64 runas, `«»`).
- [ ] 3.4 Crear `internal/changestart/untrusted.go` con `neutralize` e `inlineNeutral` (D-10, D-28). GREEN.
- [ ] 3.5 RED: crear `internal/changestart/render_test.go` con `TestRenderHeaderFirstLine`, `TestRenderRelations`, `TestRenderOmitsEmptySections`, `TestRenderTrustMetadataOutsideBlock` (`external_ref`, `plugin`, `id`, `url`, `perfil`, `importado` fuera del bloque `AXIOM::INICIO-DATOS`). Crear `internal/changestart/render.go` con la composición de `work-item.md` en tres partes. GREEN.
- [ ] 3.6 Ejecutar `go test ./internal/changestart/... -run 'TestConvert|TestNeutralize|TestInlineNeutral|TestRender'` y un fuzz acotado `go test ./internal/changestart/htmlmd -run '^$' -fuzz FuzzConvert -fuzztime 20s`; `go vet`, `gofmt -l`; commit.

Cubre: prov:74:10, prov:74:11 (amenaza de inyección de instrucciones, matriz específica).

---

## T4. Adjuntos

Commit: `feat(changestart): importar adjuntos con lista blanca, firmas, presupuesto y deduplicación`. Ruta: inline. Dependencias: T2, T3.

- [ ] 4.1 RED: crear `internal/changestart/signature_test.go` con `TestSignatureTable` (PNG, JPEG, GIF, WEBP, BMP, PDF, DOCX/XLSX/PPTX distinguidos, OLE, texto UTF-8, UTF-16 omitido) y `TestOOXMLCheckDoesNotDecompress`. Crear `internal/changestart/signature.go` (D-27). GREEN.
- [ ] 4.2 RED: crear `internal/changestart/dedup_test.go` con la tabla de la clave normalizada de D-26 (esquema y host en minúsculas, puerto por defecto, ruta plegada, `api-version`/`fileName`/`download` descartados, URL no analizable). Crear `internal/changestart/dedup.go`. GREEN.
- [ ] 4.3 RED: crear `internal/changestart/attachments_test.go` con `TestAttachmentAllowed`, `TestAttachmentExtensionRejected` (`.sh`, `.ps1`, `.exe`, `.html`), `TestAttachmentSVGRejected`, `TestAttachmentNamedLikeArtifactStaysInSubdir` (`tasks.md`, `spec.md`), `TestAttachmentContentTypeMismatch` (`text/html` con `.png`), `TestAttachmentOctetStreamWithMatchingSignature`, `TestAttachmentOctetStreamHTMLBody`, `TestAttachmentNoContentTypeIsOctetStream`, `TestImageSkippedWithoutRequest/{data,relative,file,ftp,http}` (cero peticiones y `[imagen omitida: …]` sin la URL), `TestImageDedupSingleDownload`, `TestAttachmentAndImageSameURLDedup`, `TestAttachmentOverTenMiBNoContentLength`, `TestAttachmentTotalBudget`, `TestAttachmentFailureDoesNotAbort`, `TestImageForeignHostNoRequest`, `TestAttachmentSameOrgAuthenticated`, `TestAttachmentOtherOrgSkipped`, `TestAttachmentHTTPAndUserinfoSkipped`, `TestAttachmentRedirectNotFollowed`, `TestAttachmentNamesSanitized`, `TestAttachmentNameCollisions`.
- [ ] 4.4 Crear `internal/changestart/attachments.go` (D-19): orden `AttachedFile` primero, tope de 100 candidatos, prefiltro sin petición, extensión y tamaño declarado antes de descargar, límite `min(10 MiB, presupuesto)`, comprobaciones posteriores (Content-Type, tipos nunca aceptados, firma), imágenes sin `fileName` por firma, nombres saneados (≤ 48, colisiones `-2`, `-3`), permisos `0o644`. GREEN.
- [ ] 4.5 Ejecutar `go test ./internal/changestart/... -run 'TestAttachment|TestSignature|TestImage|TestOOXML'`, `go vet`, `gofmt -l`; commit.

Cubre: prov:74:12, prov:74:13 (nombres saneados y PAT restringido), amenazas de bombas de tamaño, exfiltración del PAT y rutas con apariencia de documentación.

---

## T5. Fase 1 y CLI

Commit: `feat(cli): axiom change start --from con staging atómico bajo el common dir`. Ruta: inline por defecto; delegada (un escritor acotado, un solo cambio de comportamiento) solo si se cruza el respaldo de contexto. Dependencias: T1-T4.

- [ ] 5.1 RED: crear `internal/changestart/name_test.go` con `TestDeriveNameCanonical`, `TestDeriveNameAccentsAndSigns`, `TestDeriveNameTruncated`, `TestDeriveNameEmptySlug`, `TestDeriveNameGitDangerous`, `FuzzDeriveName` (siempre supera `ValidChangeName`, ≤ 48). Crear `internal/changestart/name.go` con `DeriveName` y `ValidChangeName` (D-08, D-28). GREEN.
- [ ] 5.2 Crear `internal/changestart/refusal.go` (`Refusal{Code, Fields, Err}`) y `internal/changestart/tracker.go` (interfaz `Tracker`, D-01, D-18).
- [ ] 5.3 RED: crear `internal/changestart/closed_test.go` y `internal/changestart/links_test.go` con `TestStartLinked/{tag_other_open,local_link_no_tag,odd_local_adoption,tag_archived_accepted,tag_archived_on_main_worktree,odd_finished_accepted,odd_registry_open,tag_unknown_locally_refused,same_name_tag,same_name_tag_case_insensitive,tag_invalid_component_ignored,tag_archive_reserved_ignored,unreadable_registry_warns}` y `TestStartLinkedTagWinsOverLocal`, sobre árboles temporales con archivo SDD, documento ODD y registros sembrados. Crear `internal/changestart/closed.go` y `internal/changestart/links.go` (D-11).
- [ ] 5.4 Crear `internal/changestart/commondir.go` (D-23, `gitcommon.GitRootFromFiles` y `CommonDirFromFiles`), `internal/changestart/lock.go` (cerrojos del WI y del cambio con `fileprim.AcquireLock`, 100 ms durante 5 s, `change_start_busy`) y `internal/changestart/sweep.go`. Pruebas: `TestStartConcurrentSameItem`, `TestStartConcurrentSameItemDifferentTitles`, `TestStartOrphanLockRecovered`, `TestStartBusyAfterBoundedWait`, `TestSweepOnlyUnderChangeLock` en `internal/changestart/lock_test.go`.
- [ ] 5.5 RED: crear `internal/changestart/start_test.go` con `TestStartStagesWithoutTouchingRepo`, `TestStartReadsTrackerOnce`, `TestStartExistingStagingSameItem`, `TestStartAlreadyStagedOtherName`, `TestStartReplacesStagingWithoutJSON`, `TestStartStagingConflictOtherItem`, `TestStartWriteFailureLeavesNoStaging` (fallo en cada escritura y en el `rename`), `TestStartAlreadyAdoptedSameChange`, `TestStartFromLinkedWorktreeUsesCommonDir`, `TestStartRelativeCwd`, `TestStartRefusals/{not_found,type_unmapped,closed_completed,closed_removed,category_not_name,unavailable_503,unavailable_timeout,unavailable_401}`, `TestStartTypeCheckedBeforeStates`, `TestStartStatesFailureIsUnavailable/{503,404}`, `TestStartBatchNotFoundOnlyWarns`, `TestStartUnknownStateWarns`, `TestStartProfileIncrement`, `TestStartProfileBugCaseInsensitive`, `TestStartProfileOtherDefectTypeIsIncrement`, `TestStartProfileBugWritesNoObservedExpected`.
- [ ] 5.6 Crear `internal/changestart/start.go`, `internal/changestart/validate.go` (tipo, perfil literal `Bug`, cierre por categoría) y `internal/changestart/staging.go` (rutas `<common>/axiom/change-start/v1/`, `work-item.json` en último lugar, publicación y retirada por `os.Rename`, costuras `Env.FS`) siguiendo el orden de D-12 y D-13. GREEN de 5.3-5.5.
- [ ] 5.7 RED: crear `internal/cli/change_start_test.go` con `TestChangeStartExclusiveModesUsage`, `TestChangeStartAdoptDiscardRequireChange`, `TestChangeStartLaneOnlyWithAdopt`, `TestChangeStartUnknownFlagAndPositionalUsage`, `TestChangeStartCheckOrder`, `TestChangeStartGitRepositoryRequired/{from,adopt,discard}`, `TestChangeStartJSONSchema`, `TestChangeStartRunsNoGit` (con `gitcommon.Observe`), `TestChangeStartNotApplicable/{no_integrations,disabled,no_axiom_yaml,invalid_yaml_warns_validate,incomplete_block_warns_validate,not_registered_warns}` (observador de red y de ficheros, en los tres modos), `TestChangeStartPortUnsupported`, `TestChangeStartWorkItemRequired`, `TestChangeStartEmptyFrom`, `TestChangeStartReferenceInvalidNoNetwork`, `TestChangeStartMissingPATNoNetwork`, `TestChangeStartNoPATInOutput`, `TestChangeStartExternalRefUsesManifestAlias`, `TestChangeStartTrackerTextConfined`, `TestChangeStartEndToEndLocalOnly`.
- [ ] 5.8 Crear `internal/cli/change_start.go`: `RunChangeStart` con el orden de comprobación de D-17, `resolveWorkItemTracker`, catálogo cerrado de rechazos como centinelas `errors.New` (cada uno nombra un `axiom …` o lleva `// refusal:by-design <forma>: <motivo>`), `changeStartRefusalJSON`, salida `axiom.change-start/v1` y texto confinado (D-18, D-20, D-28). GREEN de 5.7 en modo `--from`, consulta y rechazos de uso.
- [ ] 5.9 RED: crear `cmd/axiom/change_start_help_test.go` con `TestChangeHelpListsStart` y `TestAxiomHelpMentionsChangeStart`. Modificar `cmd/axiom/main.go`: `case "start"` en `runChange` → `cli.RunChangeStart`, ayuda de `axiom change` con `start` y sus modos, error de subcomando con `[create|new|start]`, línea `change start` en `axiomHelpText()`. GREEN.
- [ ] 5.10 Modificar `axiom.example.yaml` y `docs/manual-de-inicio.md`: `item_keys`, `ref_alias` (informativo), aviso del perfil `bug` literal (D-07), formas de `--from` (D-29), flujo `axiom change start` y nota de los ficheros de cerrojo de cero bytes (D-24).
- [ ] 5.11 Ejecutar `go test ./internal/changestart/... -run 'TestStart|TestDeriveName|FuzzDeriveName|TestLock|TestSweep|TestStartLinked'`, `go test ./internal/cli -run 'TestChangeStart'`, `go test ./cmd/axiom -run 'TestChangeHelpListsStart|TestAxiomHelpMentionsChangeStart'`, `go vet` y `gofmt -l`; commit.

Cubre: prov:74:1, prov:74:2, prov:74:3, prov:74:4, prov:74:5, prov:74:6, prov:74:7, prov:74:8, prov:74:9, prov:74:17 (cerrojos de la fase 1), prov:74:19.

---

## T6. Fase 2 y `--discard`

Commit: `feat(cli): axiom change start --adopt y --discard con copia verificada y enlace de vuelta`. Ruta: inline por defecto; delegada solo por el respaldo de contexto. Dependencias: T5.

- [ ] 6.1 RED: crear `internal/state/project_ignore_test.go` (ampliar) con `TestEnsureChangeContextIgnoredIdempotent`, `TestEnsureChangeContextIgnoredKeepsUserLines`, `TestEnsureChangeContextIgnoredWorkspacePrefix`. Modificar `internal/state/project_ignore.go`: `EnsureChangeContextIgnored(workspaceRoot string) error` y helper compartido `ensureExcludeEntries(header, entries)` que refactoriza `EnsureProjectStateIgnored` (D-16).
- [ ] 6.2 RED: ampliar las pruebas de `internal/docpaths/docpaths_test.go` con `TestLocalStateIncludesContext` (`<cambio>/context/**` y `<cambio>/.context-tmp/**`, activa y archivada). Modificar `internal/docpaths/docpaths.go` (`Set.LocalState` y `LocalStateGlobs`). Añadir `TestPublishExcludesContext` y `TestPublicationStateIgnoresContext` en `internal/sddpublish/` (puerto Git de prueba) y `TestStatusWithContextDir` en `internal/sddstatus/`.
- [ ] 6.3 Modificar `.gitignore` con las cuatro reglas (`openspec/changes/*/context/`, `openspec/changes/*/.context-tmp/`, `openspec/changes/archive/*/context/`, `odd/context/`); prueba `TestRepoGitignoreIgnoresContext` (`git check-ignore`, incluidos los temporales).
- [ ] 6.4 RED: crear `internal/changestart/copy_test.go` y `internal/changestart/adopt_test.go` con `TestAdoptSDD`, `TestAdoptODDNoIntegrationsYAML`, `TestAdoptTwiceAlreadyAdoptedNoNetwork` (SDD y ODD), `TestAdoptInterruptedRetry`, `TestAdoptDestinationOtherItemConflict`, `TestAdoptDestinationWithoutJSONConflict`, `TestAdoptWithoutStaging`, `TestAdoptChangeNotPrepared`, `TestAdoptLaneMismatch/{sdd_default_odd_record,odd_on_sdd_record,odd_head_other_branch}`, `TestAdoptHeadOtherBranch`, `TestAdoptRefusesMainCheckout`, `TestAdoptStagingIntactOnFailure/{copy,verify,rename,link,exclude,workitem_json}`, `TestAdoptStagingRetireFailureWarns`, `TestAdoptCopyAcrossVolumesEquivalent`, `TestAdoptMultirepoSpecsInPlace` (fixture multirepo del motor si existe; si no, registro sembrado a mano), `TestAdoptConcurrentWithDelivery`, `TestAdoptContextNeverUntracked` (`git status --porcelain` tras cada fallo inyectado y tras el éxito), `TestLockOrderPhase2NeverTakesItemLock`.
- [ ] 6.5 Crear `internal/changestart/copy.go` (copia a temporal en el mismo árbol y volumen, `0o644`/`0o755`, sin seguir enlaces, verificación de rutas, tamaño y SHA-256, `os.Rename`, D-15) y `internal/changestart/adopt.go` (preparación y carril incongruente D-14; pasos 1-8 de D-12 con fallos por paso de D-25; `LinkWorkItem` solo en SDD; `EnsureChangeContextIgnored`; `work-item.json` con `lane`, `adopted_at`, `backlink`; retirada del staging; `LinkBack` con `Allow` que reutiliza la regla de D-11 y D-28; reintento e idempotencia). GREEN de 6.4.
- [ ] 6.6 RED: crear `internal/changestart/linkback_adopt_test.go` (o ampliar `adopt_test.go`) con `TestAdoptTagAddedOnce`, `TestAdoptTrackerDownWarnsExitZero`, `TestAdoptRetryCompletesPendingBacklink`, `TestAdoptOtherTagAppearedBetweenPhases`, `TestAdoptArchivedTagToleratedInPhase2`, `TestAdoptRevRaceKeepsTags`; GREEN con el paso 7 de `adopt.go`.
- [ ] 6.7 RED y GREEN de `--discard`: crear `internal/changestart/discard_test.go` con `TestDiscardExisting`, `TestDiscardNothingWarns`, `TestDiscardRenameFailureWriteFailed`, `TestDiscardSweepsTemps`; crear `internal/changestart/discard.go` (D-12, D-25).
- [ ] 6.8 Ampliar `internal/cli/change_start.go` con los modos `--adopt` y `--discard`, estados `adopted`, `already_adopted`, `discarded`, `nothing_to_discard`, avisos y rechazos de la fase 2; pruebas en `internal/cli/change_start_test.go`: `TestChangeStartDiscardTraversalUsage` y recorridos extremo a extremo de `--adopt` y `--discard` con `adotest` y git temporal.
- [ ] 6.9 Prueba del motor, con git real: añadir `TestBranchDoneIgnoresIgnoredContext` en `internal/branch/engine/done_test.go` para validar que `git worktree remove` no considera sucios los ficheros ignorados de `context/` (pregunta abierta del diseño); si falla en la versión de git local, registrar el hallazgo en la propia tarea y detener.
- [ ] 6.10 Ejecutar `go test ./internal/changestart/... -run 'TestAdopt|TestDiscard|TestLockOrder'`, `go test ./internal/state ./internal/docpaths ./internal/sddpublish ./internal/sddstatus -run 'TestEnsureChangeContextIgnored|TestLocalStateIncludesContext|TestPublishExcludesContext|TestPublicationStateIgnoresContext|TestStatusWithContextDir'`, `go test ./internal/integrations -run 'TestLinkWorkItem'`, `go test ./internal/cli -run 'TestChangeStart'`, `go test ./internal/branch/... -run 'TestBranchDoneIgnoresIgnoredContext'`; `go vet`, `gofmt -l`; commit.

Cubre: prov:74:14, prov:74:15, prov:74:16, prov:74:17 (cerrojo de la fase 2 y del descarte), prov:74:18, RF-INT-09 (MOD), prov:74:19.

---

## T7. Skills y convención común

Commit: `docs(skills): tratar el contexto del work item como dato no fiable y condicionar external_ref al vínculo`. Ruta: inline. Dependencias: ninguna de código; se ejecuta antes de T8. Solo existe una copia de `sdd-phase-common.md` (`internal/assets/skills/_shared/`); no hay copias por agente que sincronizar.

- [ ] 7.1 RED: crear `internal/components/workitem_skills_test.go` con `TestSkillsTreatWorkItemContextAsUntrusted` (las cuatro fuentes: `sdd-explore`, `sdd-propose`, `sdd-iso-convention`, `sdd-phase-common` sección H) y `TestProposeExternalRefRule` (con vínculo escribe `external_ref: <alias>:<id>` tomado de `context/work-item.json` tras comprobar el id; bug con vínculo escribe `Referencia:` bajo `## Informe de bug`; sin vínculo ni `external_ref` ni `Referencia:` aunque exista `context/`; pregunta lo que falte). Observarlas fallar.
- [ ] 7.2 Modificar `internal/assets/skills/_shared/sdd-phase-common.md`: añadir al final la sección «H. Imported work-item context» (D-21).
- [ ] 7.3 Modificar `internal/assets/skills/sdd-explore/SKILL.md` (recibe `context/work-item.md` como dato no fiable y cita la sección H).
- [ ] 7.4 Modificar `internal/assets/skills/sdd-propose/SKILL.md` (la condición es el vínculo `kind: work_item` en `integrations.yaml`, no la carpeta; `external_ref` y `Referencia:`; `Observado:` y `Esperado:` desde el contexto).
- [ ] 7.5 Modificar `internal/assets/skills/_shared/sdd-iso-convention.md` (regla repetida y cita de la sección H; `Referencia:` del bug con vínculo).
- [ ] 7.6 Añadir `TestStatusDoesNotRequireExternalRef` en `internal/sddstatus/status_test.go` (un cambio sin `external_ref` no bloquea el estado).
- [ ] 7.7 Ejecutar `go test ./internal/components/... -run 'TestSkillsTreatWorkItemContextAsUntrusted|TestProposeExternalRefRule'` y `go test ./internal/sddstatus/... -run 'TestStatusDoesNotRequireExternalRef'`; si las skills participan en goldens, **no** regenerarlos aquí (se hace una sola vez en T8); `gofmt -l`; commit.

Cubre: prov:74:26, prov:74:27, prov:74:11 (lado skills).

---

## T8 (última, única). Doctrina del orquestador y goldens

Commit: `feat(doctrine): regla del work item de origen en SDD y ODD con goldens regenerados`. Ruta: inline. Dependencias: T7 (y T5-T6 para que los comandos citados existan). Esta es la única tarea que regenera goldens.

- [ ] 8.1 RED: crear `internal/components/workitem_doctrine_test.go` con `TestOrchestratorWorkItemRuleParity` (puntos 1-11 de D-21 por runtime: consulta, pedir el WI, orden fase 1 → `ensure` → `seal` → fase 2 → `proposal.md`/explore, fallo de `ensure`/`decision_required`/`seal` sin fase 2 y con reintento o `--discard`, rechazo de `--adopt` que detiene, no ejecutar `--discard` por cuenta propia, aviso que no detiene, ISO/bug, sin cambios sin plugin), `TestODDRoutingWorkItemRule` (fase 1 antes de `ensure --lane odd`, fallo de `ensure` sin fase 2, fase 2 antes de `odd/tasks`, rechazo de `--adopt` que detiene, `external_ref`, perfil bug con `Tipo: bug` y `## Spec viva` o `## Impacto en la spec`/`Sin impacto en la spec`/`Motivo:`, el verbo no crea el documento, dato no fiable, trabajo pequeño) y `TestRoutingAddsNoStop` (RF-RUT-18). Observarlas fallar.
- [ ] 8.2 Modificar `internal/assets/skills/_shared/sdd-orchestrator-sections.md`: regla del WI (en inglés) antes del párrafo de `axiom branch ensure --change` (D-21, puntos 1-11).
- [ ] 8.3 Modificar `internal/components/agentguidance/routing.go`: subsección «Work item de origen» antes de «Branches and worktrees» (puntos 1-5, 7-9 y 11 adaptados, `external_ref`, perfil bug en ODD, trabajo pequeño).
- [ ] 8.4 Regenerar goldens: `go test ./internal/components -run TestGolden -update`; revisar el diff completo de `testdata/golden/*` (solo deben cambiar las secciones de doctrina y skills esperadas); relanzar `go test ./internal/components -run TestGolden` sin `-update` y confirmar que pasa.
- [ ] 8.5 Ejecutar `go test ./internal/components/... -run 'TestOrchestratorWorkItemRuleParity|TestODDRoutingWorkItemRule|TestRoutingAddsNoStop|TestSkillsTreatWorkItemContextAsUntrusted|TestGolden'`, `gofmt -l`; commit.

Cubre: prov:74:24, prov:74:25, prov:74:26 (goldens y paridad por runtime).

---

## T9. Cierre: trinquetes antes de la PR

Commit: `chore(change-start): cerrar trinquetes de código muerto y rechazos` (solo si hay algo que ajustar; sin cambios, no hay commit). Ruta: inline.

- [ ] 9.1 `scripts/deadcode-ratchet.sh`: sin entradas nuevas en `.deadcode-baseline.txt` (D-22); todo símbolo nuevo alcanzable desde `cmd/axiom` → `cli.RunChangeStart`.
- [ ] 9.2 `go test ./internal/cli -run TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`: cada centinela del catálogo nombra un `axiom …` o declara `// refusal:by-design`.
- [ ] 9.3 Confirmar que el canario de secretos `internal/dashboard/secrets_canary_test.go` no aplica (el verbo no añade rutas al servidor del dashboard, D-22); no se ejecuta ni se modifica.
- [ ] 9.4 `go vet` y `gofmt -l` sobre los paquetes tocados: `./internal/integrations/...`, `./internal/changestart/...`, `./internal/cli`, `./internal/state`, `./internal/docpaths`, `./internal/workspace`, `./internal/components/...`, `./cmd/axiom`.
- [ ] 9.5 Comprobar con `git -C C:\repos\axiom-wt\arranque-desde-wi\axiom status --porcelain` que no quedan ficheros sin confirmar ni `context/` visible, y que `go.mod` y `go.sum` no cambian.
- [ ] 9.6 Registrar la evidencia (comandos y resultados observados) en la propia tarea y pasar a `sdd-verify`.

Cubre: prov:74:19, restricciones D-22 (alcanzabilidad), trinquete de rechazos.

---

## Orden y paralelismo

Secuencial, un solo escritor. T0 y T1 son independientes (T0 es solo documentación). T1 → T2 → T3 → T4 → T5 → T6 es una cadena de dependencias de código. T7 solo toca texto y podría ir en paralelo con T3-T6 (superficie de edición disjunta), pero se ejecuta antes de T8 por el riesgo de conflictos con C-07-e y C-07-f en `sdd-phase-common.md`. T8 es la única y última tarea que regenera goldens; T9 cierra antes de la PR.

## Cobertura de requisitos

| Requisito | Tareas |
|-----------|--------|
| prov:74:1 | T0, T5 |
| prov:74:2, prov:74:3 | T5 |
| prov:74:4 | T0, T2, T5 |
| prov:74:5, prov:74:6, prov:74:8 | T5 |
| prov:74:7 | T0, T5 |
| prov:74:9 | T0, T1, T5 |
| prov:74:10, prov:74:11 | T3, T7 |
| prov:74:12 | T4 |
| prov:74:13 | T0, T2, T4 |
| prov:74:14 | T0, T6 |
| prov:74:15, prov:74:16, prov:74:18 | T6 |
| prov:74:17 | T0, T5, T6 |
| prov:74:19 | T2, T5, T6, T9 |
| prov:74:20, prov:74:21, prov:74:22, prov:74:23 | T2 |
| prov:74:24, prov:74:25 | T8 |
| prov:74:26 | T7, T8 |
| prov:74:27 | T7 |
| RF-INT-09 (MOD) | T2, T6 |
| RF-ADO-01, RF-ADO-02 (MOD) | T1 |
| RF-ADO-16 (MOD) | T2 |

## Registro de remediación

(vacío)
