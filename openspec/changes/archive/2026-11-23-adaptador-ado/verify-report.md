# Informe de verificación: adaptador Azure DevOps (C-07-c)

Cambio `adaptador-ado`, rol único `fullstack`. Alcance: diff `main..inc/adaptador-ado` (8 commits). Informe del rol y del cambio a la vez.

## Resumen

Todos los requisitos REQ-45.1 a 45.17, MODIFIED REQ-43.4, REQ-43.18 y REQ-43.21 tienen implementación y pruebas concretas que pasan. Sin hallazgos CRITICAL ni WARNING. 1 SUGGESTION.

## Comandos ejecutados

| Comando | Resultado |
|---|---|
| `go test -count=1 ./internal/integrations/...` | ok (todos los paquetes, incluidos azuredevops, adotest, identitystore, manifest, builtin, secrets) |
| `go test -count=1 ./internal/workspace ./internal/hub ./cmd/axiom` | ok los tres |
| `go test -count=1 -timeout 45m ./internal/cli -run 'Integrations\|ADO\|Gate\|Ensure\|Publish'` | ok (803 s) |
| `go vet ./internal/integrations/... ./internal/workspace ./internal/cli ./cmd/axiom` | sin salida, exit 0 |
| `gofmt -l internal cmd` | sin ficheros |
| `scripts/deadcode-ratchet.sh` | «no new unreachable functions», exit 0 |

No se compiló ningún binario. Ningún fallo ambiental apareció (no se ejecutaron los paquetes conocidos como ajenos).

## Cumplimiento de requisitos (escenario → prueba)

- REQ-45.1 manifiesto: `manifest/azuredevops_test.go:TestAzureDevOpsManifestIsValid`, `…DeclaresOnlyDeliveredCapabilities`, `…SecretAndIdentity`; `azuredevops/plugin_test.go:TestManifestMatchesCatalog`.
- REQ-45.2 configuración y `pat`: `TestAzureDevOpsManifestConfigKeys`, `TestPATIsResolvedOnEveryRequest`, `TestPATComesOnlyFromTheSecretSource`, `TestMissingSecretIsTypedAndSendsNothing`, `workspace/integrations_test.go:TestDecodeIntegrationsAzureDevOpsWithDefaultCatalog`.
- REQ-45.3 cliente: `TestRequestCarriesAPIVersionAuthAndUserAgent`, `TestTimeoutPerAttemptIsATypedNetworkError`, `TestCancellationDuringAWaitSendsNoMoreRequests`, `TestURLEscapesSegmentsAndKeepsTheHost`.
- REQ-45.4 reintentos (`retry_test.go`): `TestRetryAfterSecondsOn429WaitsAtLeastThatLong`, `TestRetryAfterOn200PacesTheNextRequest`, `TestRetryOn503WithExponentialBackoff`, `TestExhaustedBudgetOn429ReportsAttempts`, `TestRateLimitHeadersAreExposedAndResourceIsIgnored`, `TestDroppedConnectionIsRetried`.
- REQ-45.5 errores tipados: `TestAuthErrorMessageNamesScopeAndNoRetirementDates`, `TestLoginPageIsNeverParsedAsJSON`, `TestListRepositoriesForbiddenNamesVsoCode`, `TestHugeErrorBodyIsBounded`, `TestServerEchoOfCredentialsIsRedacted`, `TestOversizedSuccessBodyIsAnInvalidResponse`.
- REQ-45.6 work items: `workitems_test.go:TestGetWorkItemExpandsRelations`, `…SplitsIntoBatchesAndKeepsTheRequestedOrder`, `…NotFoundNamesTheIDAndIsNotRetried`, `TestGetWorkItemsEmptyListSendsNothing`.
- REQ-45.7 `test /rev`: `patch_test.go:TestUpdateWorkItemWithCurrentRev`, `…WithStaleRevMakesOneRequestAndChangesNothing`, `TestEveryRejectionOfTheRevTestIsAConflict`, `TestAnUnrelated400IsNotAConflict`, `TestLostResponseThenRetryIsAConflictAndAppliesOnce`, `TestUpdateWorkItemWithoutRevOrWithCallerTestSendsNothing`.
- REQ-45.8 repositorios: `repos_test.go:TestListRepositoriesReturnsNamesAndGUIDs`, `…SecondCallComesFromTheCache`, `…FailureIsNotCached`.
- REQ-45.9 discovery: `TestDiscoverReturnsTheWholeProcess`, `TestDiscoverCopiesTheStateCategoryUntouched`, `TestDiscoverPagesMembersUntilAnIncompletePage`, `TestDiscoverAcceptsEmptyIterationsInValuesOrValue`, `TestDiscoverMemberAuthFailureFailsTheWholeDiscoveryNamingVsoProject`.
- REQ-45.10 identidad: `identity_test.go:TestResolveIdentityFromConnectionDataDoesNotQueryMembers`, `…ListsCandidatesWhenConnectionDataIsUnusable`, `…WithUserSkipsConnectionData`, `…WithUnknownUserNamesTheUserAndCandidates`.
- REQ-45.11 fichero de identidad: `identitystore/store_test.go:TestFirstPutCreatesTheFileWithTheEntry`, `TestPutKeepsOtherOrganizationsUntouched`, `TestPutReplacesTheEntryOfThatOrganizationOnly`, `TestFailedWriteThrough…`, `TestPutNeverTouchesWorkspacesJSON`, `TestCorruptFileIsReportedAndNeverOverwritten`.
- REQ-45.12 `connect`: `cli/integrations_test.go:TestIntegrationsConnectSucceedsAndStoresTheIdentity`, `…WithoutUsableIdentityListsCandidatesAndSavesNothing`, `…WithUserSavesTheMemberSelection`, `…RejectsUnconfiguredOrDisabledPlugins`, `…RequiresPlugin`, `TestIntegrationsConnectAndDiscoverHelp`, `TestIntegrationsReconnectReplacesTheEntryAndSaysSo`.
- REQ-45.13 `discover`: `TestIntegrationsDiscoverJSONAndItWritesNothing`, `…DiscoverReadable`, `…DiscoverAuthFailureNamesTheScope`.
- REQ-45.14 `Deliver`: `plugin_test.go:TestDeliverIsPermanentAndMakesNoRequest`; verbos de despacho sin cambios cubiertos por `cli/integrations_events_test.go` (suite en verde).
- REQ-45.15 PAT: `cli/integrations_ado_canary_test.go:TestADOCanaryNeverAppearsInOutputsOrFiles`, `…ServerEchoIsRedactedNotDropped`, `…ScannerDetectsAPlantedValue`; `TestTransportCauseEchoingThePATIsRedacted`.
- REQ-45.16 hermeticidad: `adotest` (`TestHTTPClientRejectsNonLocalHosts`, `TestClockSleepDoesNotSleepAndAdvancesVirtualTime`).
- REQ-45.17 puertos: `ports_test.go:TestPortsAreDetectedByTypeAssertion`, `TestPortsUnsupportedBeforeAnySecretIsResolved`; `cli/integrations_ports_test.go`; `registry_test.go` (el puerto `Plugin` no cambia).
- MOD REQ-43.4: `registry_test.go`, `builtin_test.go:TestDefaultRegistryMatchesCatalog`, `manifest/catalog_test.go`.
- MOD REQ-43.18: `cmd/axiom/integrations_help_test.go`, `cli/integrations_test.go:…Help`.
- MOD REQ-43.21: `workspace/docs_contract_test.go`, `workspace/example_config_test.go` (manual y `axiom.example.yaml`).

Limitación: la correspondencia está hecha por nombre y alcance de prueba, sin ejecutar cada escenario por separado; no se detectan escenarios sin prueba.

## Criterio de terminado (C-07-c)

- Conecta con PAT y resuelve identidad: cumplido (REQ-45.10/45.12).
- Discovery completo: cumplido (REQ-45.9/45.13).
- Reintento ante 429 y `Retry-After`: cumplido (REQ-45.4).
- `/rev` obsoleto sin pisar datos: cumplido (REQ-45.7).
- PAT solo en el almacén y ausente de logs: cumplido (REQ-45.15, canario en el flujo completo).

## Conformidad con el diseño

D-01 a D-12 reflejados en el código (cliente, parche con `test /rev`, caché por proceso, puertos opcionales, `identitystore`, verbos, `adotest`). Las desviaciones anotadas (scope `vso.work` en iteraciones; lista de verbos de REQ-43.18) siguen a la spec y están respaldadas por `TestDiscoverIterationAuthFailureNamesVsoWork`. No cuentan como incumplimiento.

## Tareas

45 casillas marcadas, con evidencia observada en las pruebas anteriores. Sin marcar: P.4 (confirmar que `REQ-45` está libre en `main` antes de archivar; es una comprobación de archivo, no de `apply`).

## Cumplimiento técnico

- Build/vet limpio: `go vet` exit 0.
- Estilo: `gofmt -l` vacío.
- Código muerto: ratchet sin funciones nuevas inalcanzables.
- Sin llamadas reales: `git grep` de `dev.azure.com`/`vssps` solo en el endpoint por defecto (`client.go:26`), manifiesto, documentación y `adotest`; pruebas con `httptest`, cliente que rechaza hosts no locales.
- Sin secretos reales: búsqueda de cadenas tipo PAT en `azuredevops` sin resultados.
- `go.mod`/`go.sum` y `internal/hub/manager.go`: diff vacío.
- Avisos de contexto: ninguna ruta de la lista cerrada tocada (no hay manifiestos de dependencias, `Makefile` ni CI en el diff).

## Hallazgos

- CRITICAL: ninguno.
- WARNING: ninguno.
- SUGGESTION: P.4 sigue abierta; resolverla al archivar.

**Estado:** limpio

## Judgment-day (2026-11-22)

- **Objetivo congelado:** commit `43321506` (árbol `c531c50d`), diff de código `a8cb13b4..43321506` sin `openspec/`.
- **Jueces:** `jd-judge-a` y `jd-judge-b`, ciegos y en paralelo. Los dos devolvieron un resultado utilizable.
- **Hallazgos:**
  - CRITICAL: ninguno.
  - WARNING confirmados por ambos jueces (informativos):
    - **JD-1.** `azuredevops/client.go:482-493`. `parseRetryAfter` multiplica el valor de `Atoi` por `time.Second` sin acotarlo. Un `Retry-After` de unos 9,3e9 s o más desborda a una duración negativa, se salta el techo de 60 s (`client.go:328`) y el cliente reintenta con retroceso en lugar de parar con `RateLimitError`. Es determinista.
    - **JD-2.** `identitystore/store.go:118-146`. `Put` lee, modifica y escribe `integration-identities.json` sin bloqueo entre procesos, así que dos `connect` simultáneos pueden perder una entrada. Es el riesgo R5 del diseño, que comparte modelo con `hub.Manager`.
  - WARNING de un solo juez (sospechoso, no se arregla):
    - **JD-3** (B). `connectionData` se pide con `api-version=7.1`; si el servicio exige `7.1-preview`, la ruta automática siempre caería a elegir entre los miembros. No está verificado contra Azure DevOps real.
  - SUGGESTION:
    - **JD-4** (A y B). Las cabeceras `X-RateLimit-*` aceptan `NaN`, `Inf` o valores enormes y los convierten sin acotar. Solo afectan a `RateLimit()`, que es informativo.
    - **JD-5** (A y B). Los textos que vienen del tracker (displayName, id, estados) se imprimen sin quitar los caracteres de control ni las secuencias ANSI.
    - **JD-6** (A). Las respuestas cuyo cuerpo no se lee (401, 403, 404, 3xx, 203, 204) no se drenan antes de cerrarse, y la conexión no se reutiliza.
- **Rondas de arreglo:** ninguna. Solo se arreglan los hallazgos graves que confirman los dos jueces.
- **Contradicciones:** ninguna.

JUDGMENT: APPROVED ✅

### Corrección posterior de JD-1 (2026-11-22, a petición del usuario)

- **Arreglo.** `parseRetryAfter` lee con `strconv.ParseInt` y trata `ErrRange` como el extremo del rango. Cualquier valor por encima de `MaxInt64 / time.Second` se satura a `time.Duration(math.MaxInt64)`, así que nunca da la vuelta y siempre supera el techo de 60 s.
- **RED observado:**
  - `TestParseRetryAfter`: `"9223372037"` daba una duración negativa.
  - `TestRetryAfterBeyondTheCeilingIsNotRetried/429_con_un_valor_que_desbordaría_time.Duration`: se enviaban 4 intentos.
- **GREEN:**
  - `go test -count=1 ./internal/integrations/...`: ok.
  - `go vet ./internal/integrations/azuredevops`: limpio.
  - `gofmt -l internal/integrations`: vacío.
- **Prueba añadida:** el ritmo tras un 200 con ese valor no envía ni espera (`TestPacingBeyondTheCeilingSendsNothing/10000000000`).
- **Pendientes informativos:** JD-2 a JD-6 siguen abiertos.
