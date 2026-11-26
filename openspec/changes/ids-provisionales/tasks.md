# Tareas: IDs provisionales, número asignado al integrar y declaración de impacto en `status` (C-03-b)

Rol único `fullstack` (`kickoff.yaml`). Fuentes: `proposal.md`, las seis deltas de `specs/` (REQ-47.1 a REQ-47.10; MODIFIED REQ-21.17, REQ-30.2, REQ-30.3, REQ-30.4, REQ-30.22, REQ-29.3, REQ-29.4, REQ-24.58) y `design.md` (D1 a D12, D4b, unidades U1 a U8, tabla de resolución de cobertura).

Normas de ejecución para todo el plan:

- TDD: en cada comportamiento, primero la prueba que falla (RED observado) y después la producción (GREEN); REFACTOR solo con las pruebas en verde.
- Verificación localizada, nunca la batería completa: no se ejecuta `go test ./...` ni `go test ./internal/cli` entero; en `internal/cli` y `internal/assets` siempre con `-run`. Tampoco `go build -o` de binarios sueltos. La suite completa la pasa la CI.
- Cada unidad cierra con un único commit convencional, sin atribución a IA, con código, pruebas y documentación juntos.
- No hay ningún umbral de tamaño que gestionar: no se pide `size:exception`.

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | L |
| Estimated changed lines | 6.000 a 9.000 (adiciones + eliminaciones, sin ficheros generados; cerca de dos tercios son pruebas) |

### Work Units

| Unit | Goal | Commit | Focused test command | Runtime harness | Rollback boundary |
|------|------|--------|----------------------|-----------------|-------------------|
| U1 | Puerto `reqid`: provisional, forma laxa, `IDMap`, `Allocator`, máscara de `Scan`, regla de arquitectura; `NextFree` delega | `feat(reqid): forma provisional, IDMap y Allocator para asignar el siguiente número libre` | `go test ./internal/reqid/...` y `go test ./internal/livespec -run 'Registry\|NextFree'` | N/A: biblioteca sin comando propio; la cubren las pruebas de tabla y el fuzz | `internal/reqid/*` y `internal/livespec/registry.go` |
| U2 | Errores tipados de `Compose`, `ParseChangeSpec`/`Declared` y continuación en `archive-compose` | `feat(openspecdoc): error tipado de ID inexistente y rechazo del provisional en Compose` | `go test ./internal/openspecdoc/...` y `go test ./internal/cli -run 'SDDArchiveCompose'` | `axiom sdd archive-compose` en proceso sobre fixture de `t.TempDir()` | `internal/openspecdoc/*` y `internal/cli/sdd_archive_compose.go` |
| U3 | ISO con provisionales: `RequirementKeys`, REQ-29.3 y REQ-29.4, `RewriteCubre`, `RewriteMatrix` | `feat(isoevidence): aceptar el provisional en el ADDED, Cubre y la matriz` | `go test ./internal/isoevidence/...` y `go test ./internal/sddstatus -run 'ISO'` y `go test ./internal/cli -run 'SDDTrace'` | `axiom sdd trace` en proceso sobre cambio ISO de fixture | `internal/isoevidence/*`, `internal/sddstatus/iso.go`, `internal/cli/sdd_trace.go` |
| U4 | Plan único: validación previa, `PlanIDs`, `classify` compartido, traza robusta, `writeSet`, duplicados | `feat(livespec): plan único de asignación de IDs con validación previa y escritura atómica` | `go test ./internal/livespec/...` | N/A: motor sin comando propio; lo ejercitan U5 y U6 | `internal/livespec/{ids,trace,apply,plan}.go` y sus pruebas |
| U5 | ODD: reescritura del documento, reintegración D5, `--check` (D4b), informe | `feat(odd): asignar el número al integrar, reescribir el documento y añadir archive integrate --check` | `go test ./internal/livespec/... -run 'ODD\|Report\|Reassign'` y `go test ./internal/cli -run 'ArchiveIntegrate'` | `RunArchiveIntegrate` en proceso con `--odd` y `--odd --check` sobre árbol temporal | `internal/livespec/{odddoc,report}.go` y `internal/cli/archive_integrate.go` |
| U6 | SDD: `PlanChange`, verbo `axiom sdd archive-integrate`, `archive-check`, ayuda, prueba G1 | `feat(sdd): verbo archive-integrate que asigna IDs y compone todo el cambio` | `go test ./internal/livespec/... -run 'Change'` y `go test ./internal/cli -run 'SDDArchiveIntegrate\|SDDArchiveCheck'` y `go test ./internal/sddstatus -run 'Gate'` y `go test ./internal/kickoff/...` y `go test ./cmd/axiom -run 'Help'` | Verbo en proceso con `--cwd` absoluto, con y sin `--check`, sobre cambio de fixture | `internal/livespec/change.go`, `internal/cli/sdd_archive_integrate.go`, `internal/cli/sdd_archive_check.go`, `cmd/axiom/main.go` |
| U7 | `status.requirements` y contrato | `feat(sddstatus): proyectar requirements.modifies y requirements.creates en status` | `go test ./internal/sddstatus -run 'Requirements'` y `go test ./internal/assets -run 'StatusContract'` | `axiom sdd status --json` en proceso sobre cambio de fixture | `internal/sddstatus/requirements.go`, campo de `status.go` y sección del contrato |
| U8 | Guías y documentación: skills, paso 7 ODD, manual, pins, goldens | `docs(sdd): guías del provisional, paso de archivado Go y paso 7 ODD con pins` | `go test ./internal/assets -run 'SDDArchive\|SDDSpec\|ISO\|StatusContract'` y `go test ./internal/components/agentguidance -run Routing` y `go test ./internal/workspace -run DocsPin` | N/A: solo texto; los pins comprueban el contenido renderizado | Textos de skills, `routing.go`, manual y pins |

Entrega `single-pr`: los ocho commits forman una sola PR de código; la PR del archive va aparte.

## Review Workload Forecast

| Unidad | Producción (líneas) | Pruebas (líneas) | Docs y pins (líneas) | Total aproximado |
|--------|---------------------|------------------|----------------------|------------------|
| U1 | 450 a 600 | 600 a 800 | 0 | 1.050 a 1.400 |
| U2 | 250 a 350 | 350 a 450 | 0 | 600 a 800 |
| U3 | 250 a 350 | 350 a 450 | 0 | 600 a 800 |
| U4 | 700 a 900 | 900 a 1.200 | 0 | 1.600 a 2.100 |
| U5 | 450 a 600 | 600 a 800 | 30 a 60 | 1.080 a 1.460 |
| U6 | 600 a 800 | 700 a 950 | 30 a 60 | 1.330 a 1.810 |
| U7 | 200 a 300 | 250 a 350 | 60 a 100 | 510 a 750 |
| U8 | 0 | 150 a 250 | 350 a 550 | 500 a 800 |
| **Total** | **2.900 a 3.900** | **3.900 a 5.250** | **470 a 770** | **6.000 a 9.000** |

Previsión de revisión: la PR es grande; la revisión por commit (un commit por unidad, con su prueba) es el modo de lectura recomendado. No se solicita `size:exception`: el comprobador de 400 líneas está retirado.

---

## Fase 1 (U1): Puerto `reqid`

Depende de: nada. Paquetes: `internal/reqid`, `internal/livespec/registry.go`.

- [x] 1.1 RED: en `internal/reqid/provisional_test.go`, tabla de la gramática de D1: `prov:30:1`, `prov:a:1`, `prov:USU:1`, `prov:USU-3:1`, `prov:USU-b:1` y `prov::1` con los cinco patrones (`REQ-<n>.<m>`, `RF-<DOM>-<NN>`, `RF-<m>`, `HU-<NNN>`, `RF-<DOM>-<n>.<m>`); dominio no admitido por la plantilla, etiqueta con patrón con `<DOM>` y componentes de distinto número o clase dan `provisional_malformed` (`*ProvisionalError`). Prueba además que un provisional nunca es `Valid`, nunca lo devuelve `InTitle` ni `TrailingCandidate`, y que `Scan` no lo devuelve como candidato (también con `RF-<DOM>-<n>.<m>` y dominio `RF`, clave `RF-3`).
  Cubre: REQ-47.1, REQ-47.3
  Verificación: `go test ./internal/reqid -run 'Provisional'`
- [x] 1.2 GREEN: crear `internal/reqid/provisional.go` con `Provisional` (`String`, `Label`), `Pattern.ParseProvisional`, `ParseProvisionalToken`, `ProvisionalInTitle`, `ScanProvisional` con la forma laxa y la regla de token completo (ni precedido ni seguido de letra, cifra, `_` o `-`), y `ProvisionalError{Token, Reason}`. Modificar `internal/reqid/reqid.go` para que `Scan` enmascare los tokens provisionales antes de buscar candidatos (REQ-47.1.1).
  Cubre: REQ-47.1
  Verificación: `go test ./internal/reqid -run 'Provisional|Scan'`
- [x] 1.3 RED y GREEN: en `internal/reqid/allocate_test.go` y `internal/reqid/allocate.go`, `Pattern.SeriesOf`, `Allocator` (`Occupy`, `Next`, `NewSeries`) y `AllocationError`. Casos: `Next` en serie vacía da `1` con el relleno de la plantilla; `NewSeries` toma el siguiente `<n>` libre y nunca una serie ocupada ni fijada por el propio cambio; desbordamiento y formato no `Valid` dan `*AllocationError`; ocupación por definitivos repetidos no altera el máximo.
  Cubre: REQ-47.2
  Verificación: `go test ./internal/reqid -run 'Allocat|NewSeries|SeriesOf'`
- [x] 1.4 RED y GREEN: en `internal/reqid/idmap_test.go` y `internal/reqid/idmap.go`, `IDMap`, `Assignment`, `NewIDMap`, `Substitute` por token completo (el más largo primero, una sola pasada, con claves provisionales y definitivas para la reasignación D→D') y `Count`. Casos: `prov:30:1` no altera `prov:30:11`; token pegado a letra, cifra, `_` o `-` no se sustituye; sustitución idempotente.
  Cubre: REQ-47.4
  Verificación: `go test ./internal/reqid -run 'IDMap|Substitute'`
- [x] 1.5 RED y GREEN: en `internal/reqid/arch_test.go`, regla «el literal `prov:` solo aparece en código de producción de `internal/reqid`» y conservación de la prueba de hoja (solo biblioteca estándar); en `internal/reqid/fuzz_test.go`, propiedad «provisional ⇒ no válido, ni título, ni candidato».
  Cubre: REQ-47.1
  Verificación: `go test ./internal/reqid -run 'Arch|Fuzz'`
- [x] 1.6 RED y GREEN: modificar `internal/livespec/registry.go` para que `Registry.NextFree` delegue en `reqid.Allocator`; las pruebas existentes `registry_test.go` y `registry_pattern_test.go` se conservan sin cambios (regresión de comportamiento).
  Cubre: REQ-47.2
  Verificación: `go test ./internal/livespec -run 'Registry|NextFree'`
- [x] 1.7 Cierre de U1: `go vet ./internal/reqid/... ./internal/livespec/...` y commit `feat(reqid): forma provisional, IDMap y Allocator para asignar el siguiente número libre`.
  Cubre: REQ-47.1, REQ-47.2, REQ-47.4
  Verificación: `go test ./internal/reqid/...`

## Fase 2 (U2): Errores tipados de `Compose` y `Declared`

Depende de: U1. Paquetes: `internal/openspecdoc`, `internal/cli/sdd_archive_compose.go`.

- [x] 2.1 RED: en `internal/openspecdoc/compose_test.go`, `UnknownRequirementIDError` en MODIFIED, REMOVED y nombre antiguo de RENAMED con ID válido que `Resolve` no encuentra (`errors.As` también al `*UnappliedDeltaError` mediante `Unwrap()`); `ProvisionalIDError` ante un provisional en el nombre y en el cuerpo de un requisito de cualquier sección; un MODIFIED sin ID se resuelve por título y conserva el rechazo vigente si no casa; un nombre con dos definitivos conserva la regla actual.
  Cubre: REQ-47.3, REQ-30.4, REQ-30.22
  Verificación: `go test ./internal/openspecdoc -run 'Compose.*(Unknown|Provisional)'`
- [x] 2.2 GREEN: modificar `internal/openspecdoc/compose.go` con `UnknownRequirementIDError{Section, Requirement, ID}` y `ProvisionalIDError{Section, Requirement, Token}`, y los puntos de emisión de D7.
  Cubre: REQ-47.3, REQ-30.22
  Verificación: `go test ./internal/openspecdoc -run 'Compose'`
- [x] 2.3 Regresión: el corpus `compose_corpus_test.go` sigue siendo byte a byte para toda composición aceptada hasta hoy.
  Cubre: REQ-30.22
  Verificación: `go test ./internal/openspecdoc -run 'Corpus'`
- [x] 2.4 RED y GREEN: crear `internal/openspecdoc/declared.go` y su prueba con `ParseChangeSpec`, `DeclaredRequirement`, `Declaration` y `Declared(capability, delta, pattern)`. Casos: ADDED con provisional (`id` y `provisional: true`), ADDED con definitivo, MODIFIED sin ID (sin `id`), RENAMED por nombre antiguo, capacidad nueva como spec completa que cuenta como ADDED, orden estable.
  Cubre: REQ-47.9, REQ-47.7
  Verificación: `go test ./internal/openspecdoc -run 'Declared|ParseChangeSpec'`
- [x] 2.5 RED y GREEN: modificar `internal/cli/sdd_archive_compose.go` para que un `ProvisionalIDError` lleve la continuación al verbo `axiom sdd archive-integrate` y un `UnknownRequirementIDError` añada la capacidad por la carpeta; `archive-compose` conserva su comportamiento con nombres superados inexistentes (REQ-30.23).
  Cubre: REQ-30.22, REQ-47.8
  Verificación: `go test ./internal/cli -run 'SDDArchiveCompose'`
- [x] 2.6 Cierre de U2: `go vet ./internal/openspecdoc/... ./internal/cli` y commit `feat(openspecdoc): error tipado de ID inexistente y rechazo del provisional en Compose`.
  Cubre: REQ-47.3, REQ-47.9, REQ-30.22
  Verificación: `go test ./internal/openspecdoc/...`

## Fase 3 (U3): ISO con provisionales

Depende de: U1. Paquetes: `internal/isoevidence`, `internal/sddstatus/iso.go`, `internal/cli/sdd_trace.go`.

- [x] 3.1 RED: en `internal/isoevidence/reqid_test.go`, `RequirementKeys(name, pattern)` devuelve `valid`, `provisional` y `malformed`; REQ-29.3 admite un ADDED con provisional y rechaza con `iso_requirement_id` un nombre con provisional más otro identificador y un provisional fuera de ADDED.
  Cubre: REQ-29.3, REQ-47.3
  Verificación: `go test ./internal/isoevidence -run 'RequirementKeys|RequirementID'`
- [x] 3.2 GREEN: modificar `internal/isoevidence/reqid.go` para sustituir `RequirementIDs` por `RequirementKeys` en `CheckRequirementIDs`, `deltaIDs`, `CheckCoverage` y `AffectedFromDelta`; ajustar los llamadores en `internal/sddstatus/iso.go` y `internal/cli/sdd_trace.go`.
  Cubre: REQ-29.3, REQ-29.4
  Verificación: `go test ./internal/isoevidence/... ./internal/sddstatus -run 'ISO'`
- [x] 3.3 RED y GREEN: en `internal/isoevidence/cubre_test.go`, REQ-29.4 con provisional: `Cubre:` acepta un provisional declarado por un ADDED de la propia delta y un provisional no declarado da `iso_cubre_unknown` con detalle propio; cada requisito de la delta aparece en alguna línea `Cubre:`.
  Cubre: REQ-29.4
  Verificación: `go test ./internal/isoevidence -run 'Cubre|Coverage'`
- [x] 3.4 RED y GREEN: modificar `internal/isoevidence/cubre.go` y `internal/isoevidence/matrix.go` con `RewriteCubre` y `RewriteMatrix` (sustituyen solo el token, conservan posición y bytes restantes; la matriz solo en la columna `Requisito`; la clave de escenario coincide con `AffectedFromDelta`).
  Cubre: REQ-47.4, REQ-29.3, REQ-47.8
  Verificación: `go test ./internal/isoevidence -run 'Rewrite|Matrix'`
- [x] 3.5 Regresión: `internal/cli/sdd_trace.go` con provisionales en una delta de fixture ISO (traza legible, sin error espurio).
  Cubre: REQ-29.3, REQ-29.4
  Verificación: `go test ./internal/cli -run 'SDDTrace'`
- [x] 3.6 Cierre de U3: `go vet ./internal/isoevidence/... ./internal/sddstatus/...` y commit `feat(isoevidence): aceptar el provisional en el ADDED, Cubre y la matriz`.
  Cubre: REQ-29.3, REQ-29.4, REQ-47.4
  Verificación: `go test ./internal/isoevidence/...`

## Fase 4 (U4): Plan único en `livespec`

Depende de: U1 a U3. Paquete: `internal/livespec`.

- [ ] 4.1 RED: en `internal/livespec/ids_test.go`, un caso por código de D7 con el motivo y los campos esperados: `provisional_malformed`, `provisional_new_series_unsupported`, `provisional_outside_added`, `provisional_duplicate`, `provisional_ambiguous`, `provisional_undeclared`, `unknown_requirement_id` (con capacidad), `id_allocation_failed`, `superseded_requirement_unknown`, `superseded_requirement_ambiguous`, `flat_delta_unsupported`. Cada uno rechaza sin escribir.
  Cubre: REQ-47.3, REQ-30.4
  Verificación: `go test ./internal/livespec -run 'IDPlanError|PlanIDs.*Reject'`
- [ ] 4.2 GREEN: crear `internal/livespec/ids.go` con `IDPlanInput`, `CapabilitySource`, `Citation`, `IDPlan`, `IDPlanError` y `PlanIDs` con la validación previa de D3 paso 2 (a a g) en ese orden, idéntica con y sin `--check`.
  Cubre: REQ-47.3
  Verificación: `go test ./internal/livespec -run 'PlanIDs'`
- [ ] 4.3 RED y GREEN: modificar `internal/livespec/plan.go` para compartir `classify` y `reducedDelta` entre ODD y SDD: capacidad sin canónica con MODIFIED, REMOVED o RENAMED da `capability_missing`; ADDED presente con otro cuerpo da `requirement_diverged`; REMOVED que ya no se resuelve y RENAMED ya aplicado son `already`, no error; un MODIFIED sin ID que no casa sigue pendiente y lo rechaza la composición con el motivo vigente.
  Cubre: REQ-47.3, REQ-30.4, REQ-30.22
  Verificación: `go test ./internal/livespec -run 'Classify|MissingCapability|Divergent'`
- [ ] 4.4 RED y GREEN: asignación determinista (D3 pasos 1, 3 y 4): orden por capacidad y por aparición, dos capacidades consecutivas sin repetir número, ocupación por registro, ADDED definitivos propios, cambios activos (con `ExcludeChange`) y definitivos de la traza, serie nueva que no recae en una fijada, etiquetas con números consecutivos desde 1, y la entrada de la traza conserva su definitivo si cambia el orden.
  Cubre: REQ-47.2, REQ-47.3
  Verificación: `go test ./internal/livespec -run 'PlanIDs.*(Assign|Order|Occup|Series)'`
- [ ] 4.5 RED y GREEN: crear `internal/livespec/trace.go` y `trace_test.go` con `Trace` YAML (`axiom.requirement-ids/v1`), lectura, validación y escritura en fichero y en sección ODD. Casos: YAML ilegible, esquema desconocido, sección duplicada y definitivo no válido dan `id_trace_invalid`; `pattern` distinto da `id_trace_pattern_mismatch`; entrada sin exactamente un ADDED de su capacidad da `id_trace_stale`; traza borrada degrada a definitivos elegidos por el autor; sin asignaciones no se escribe.
  Cubre: REQ-47.6, REQ-47.3
  Verificación: `go test ./internal/livespec -run 'Trace'`
- [ ] 4.6 RED y GREEN: reintegración D5 en el plan: `assigned`, `kept` (D libre y D ya integrado con cuerpo igual módulo sustitución), rechazo `requirement_diverged` para el bloque propio editado (por título guardado en la traza), `reassigned` individual con serie fijada y por etiqueta entera con serie nueva, con `previous`.
  Cubre: REQ-47.5, REQ-30.3
  Verificación: `go test ./internal/livespec -run 'Reassign|Kept|Reintegr'`
- [ ] 4.7 RED y GREEN: duplicados (REQ-47.5): tras componer en memoria se cuentan las apariciones de cada ID que escribe la integración en todas las capacidades resultantes; más de una da `duplicate_assigned_id`; los históricos ajenos se informan con `blocking: false`.
  Cubre: REQ-47.5
  Verificación: `go test ./internal/livespec -run 'Duplicate'`
- [ ] 4.8 RED: en `internal/livespec/apply_test.go`, atomicidad sin temporales con fallo inyectado en cada escritura de `writeSet` (costuras `writeTempFile` y `renameFile`): ningún `.axiom-tmp` residual, árbol sin cambios; fallo de restauración conserva `rollbackError` con cada ruta no restaurada y la continuación `git checkout -- <ruta>`; temporal huérfano de un proceso matado se sustituye y se consume en la siguiente ejecución real; `--check` no lo toca; carpetas creadas se retiran.
  Cubre: REQ-47.4, REQ-30.2
  Verificación: `go test ./internal/livespec -run 'WriteSet|Apply|Rollback|Orphan'`
- [ ] 4.9 GREEN: modificar `internal/livespec/apply.go` para generalizar `Integration.Apply` a `writeSet(files []fileWrite, opts)`; `Apply` lo usa y `rollbackError` se conserva; el nombre del temporal es determinista (`.<base>.axiom-tmp`).
  Cubre: REQ-47.4
  Verificación: `go test ./internal/livespec -run 'WriteSet|Apply|Rollback|Orphan'`
- [ ] 4.10 RED y GREEN: no regresión de los lectores de deltas activas (D11): `livespec/registry.go` no indexa ni colisiona un `(prov:…)` de un cambio activo; `livingdoc/indexer.go` no lo toma como `TrailingCandidate`; el dashboard (`odd_livespec_test.go`) ignora la sección de traza; `ParseFeatureDoc` valida igual con la traza y una traza duplicada da `id_trace_invalid`.
  Cubre: REQ-47.1, REQ-47.6
  Verificación: `go test ./internal/livespec -run 'Registry.*Provisional|ParseFeatureDoc.*Trace'` y `go test ./internal/livingdoc/... -run 'Provisional'` y `go test ./internal/dashboard/... -run 'LiveSpec'`
- [ ] 4.11 Cierre de U4: `go vet ./internal/livespec/... ./internal/livingdoc/...` y commit `feat(livespec): plan único de asignación de IDs con validación previa y escritura atómica`.
  Cubre: REQ-47.2, REQ-47.3, REQ-47.4, REQ-47.5, REQ-47.6
  Verificación: `go test ./internal/livespec/...`

## Fase 5 (U5): Carril ODD

Depende de: U4. Paquetes: `internal/livespec`, `internal/cli/archive_integrate.go`.

- [ ] 5.1 RED: en `internal/livespec/odddoc_test.go`, el documento ODD se reescribe en el mismo conjunto atómico: provisional → definitivo en la valla `openspec` de `## Spec viva` y en líneas `Cubre:` fuera de vallas; el resto del documento no cambia salvo `## Identificadores asignados` (un único bloque `yaml`); un provisional citado en el resto del documento se valida pero no se reescribe.
  Cubre: REQ-30.2, REQ-47.4, REQ-47.6
  Verificación: `go test ./internal/livespec -run 'ODDDoc|Rewrite'`
- [ ] 5.2 GREEN: modificar `internal/livespec/odddoc.go` con la reescritura del documento y la sección de traza, y el orquestador `PlanIntegration` con provisionales sobre el plan único.
  Cubre: REQ-30.2, REQ-30.3, REQ-30.4
  Verificación: `go test ./internal/livespec -run 'ODD|PlanIntegration'`
- [ ] 5.3 RED: reintegración de dos ramas concurrentes con números distintos en el carril ODD: dos documentos con un ADDED provisional en la misma capacidad se integran uno tras otro y reciben números distintos; tras un rebase simulado en el que la base ocupó el definitivo con otro cuerpo, la reejecución reasigna con `previous`; documento ya integrado devuelve `kept` y `nothingToIntegrate`.
  Cubre: REQ-30.3, REQ-47.5
  Verificación: `go test ./internal/cli -run 'ArchiveIntegrate.*(Concurrent|Rebase|Idempot)'`
- [ ] 5.4 RED: `--check` de solo lectura (D4b): el hash del árbol no cambia, `INDEX.md` intacto, mismo motivo de rechazo que la ejecución real (contexto incluido, REQ-30.6), JSON `axiom.odd-integration/v1` con `check: true`, `requirements {modifies, creates}` y `assignments` con estado `assigned|kept|reassigned`; sobre un documento ya integrado código `0` y todas las entradas `kept`.
  Cubre: REQ-47.7
  Verificación: `go test ./internal/cli -run 'ArchiveIntegrate.*Check'`
- [ ] 5.5 GREEN: modificar `internal/cli/archive_integrate.go` (bandera `--check`, recorrido completo salvo `guardODDWriter` y `plan.Apply()`, ayuda y texto) y `internal/livespec/report.go` (`check`, `assignments`, `requirements`, `duplicates`); conservar `added_without_id` para un ADDED sin definitivo ni provisional, con un texto que sugiere el provisional. Actualizar la prueba de «el documento ODD no se modifica» a la nueva garantía de REQ-30.2.
  Cubre: REQ-47.7, REQ-30.2, REQ-30.4
  Verificación: `go test ./internal/cli -run 'ArchiveIntegrate'`
- [ ] 5.6 Cierre de U5: `go vet ./internal/livespec/... ./internal/cli` y commit `feat(odd): asignar el número al integrar, reescribir el documento y añadir archive integrate --check`.
  Cubre: REQ-47.7, REQ-30.2, REQ-30.3, REQ-30.4
  Verificación: `go test ./internal/livespec/...` y `go test ./internal/cli -run 'ArchiveIntegrate'`

## Fase 6 (U6): Carril SDD

Depende de: U4. Paquetes: `internal/livespec`, `internal/cli`, `cmd/axiom`, `internal/sddstatus`, `internal/kickoff`.

- [ ] 6.1 RED: en `internal/livespec/change_test.go`, `PlanChange`: capacidad existente (`composed`), capacidad nueva (`created`, y sin provisionales byte a byte igual al `cp` de hoy y sin `ids.yaml`), serie nueva con los patrones `REQ-<n>.<m>` y `RF-<DOM>-<NN>`, capacidad parcialmente integrada (`resumed` con `skipped[]`), ya integrada (`already_integrated`), capacidad nueva con MODIFIED (`capability_missing`), paridad byte a byte con `archive-compose` para toda composición aceptada hasta hoy.
  Cubre: REQ-47.8, REQ-30.22, REQ-21.17
  Verificación: `go test ./internal/livespec -run 'PlanChange'`
- [ ] 6.2 GREEN: crear `internal/livespec/change.go` con `ChangeOptions`, `PlanChange` y `ChangeIntegration.Apply` (composición con la delta sustituida entera, reanudación con `reducedDelta`, copia de capacidad nueva, escritura por `writeSet`; reescrituras de deltas, `Cubre:` de `tasks.md` y `tasks.<rol>.md`, columna `Requisito` de `verify-report*.md` y `ids.yaml`, solo con mapa no vacío).
  Cubre: REQ-47.8, REQ-47.4, REQ-21.17
  Verificación: `go test ./internal/livespec -run 'PlanChange|Change'`
- [ ] 6.3 RED: superación: `--supersede` para los REMOVED de todas las capacidades; nombres de `--superseded-requirements` con y sin prefijo `<capacidad>/`; nombre inexistente da `superseded_requirement_unknown`; nombre ambiguo da `superseded_requirement_ambiguous` con la continuación del prefijo; `archive-compose` conserva su comportamiento (REQ-30.23).
  Cubre: REQ-47.8, REQ-30.23
  Verificación: `go test ./internal/livespec -run 'Supersed'`
- [ ] 6.4 RED: `internal/cli/sdd_archive_integrate_test.go`: esquema `axiom.sdd-archive-integrate/v1` completo (`capabilities`, `requirements`, `assignments`, `duplicates`, `written`, `nothingToIntegrate`, `error`); guardas en orden (baseline residual, patrón inaccesible, negativa ISO, guarda A `guardChangeWriter` con contexto `archive`); `--check` omite solo la guarda A; guarda A en rama ajena rechaza y el árbol queda intacto; ejecución con `--cwd` absoluto desde otro directorio solo cambia ficheros del workspace; reejecución idempotente; delta plana `specs/spec.md` da `flat_delta_unsupported`; fallo inyectado sin estado a medias.
  Cubre: REQ-47.8, REQ-21.17, REQ-47.3
  Verificación: `go test ./internal/cli -run 'SDDArchiveIntegrate'`
- [ ] 6.5 GREEN: crear `internal/cli/sdd_archive_integrate.go` con el verbo `axiom sdd archive-integrate --cwd --change [--supersede] [--superseded-requirements] [--check] [--json]`, sin reutilizar la lápida `sdd integrate`.
  Cubre: REQ-47.8
  Verificación: `go test ./internal/cli -run 'SDDArchiveIntegrate'`
- [ ] 6.6 RED y GREEN: modificar `internal/cli/sdd_archive_check.go` para que `archive-check` devuelva `assignments` (de `ids.yaml`) y `pendingProvisionals`, y reconozca una asignación ya hecha tras un archivado interrumpido; `--check` sobre un cambio archivado (`changes/archive/*-<cambio>/ids.yaml`) da `duplicate_assigned_id` si un definitivo de la traza aparece más de una vez en la spec viva o fuera de su capacidad, con la continuación de D5; traza legible tras archivar.
  Cubre: REQ-47.6, REQ-47.8, REQ-47.5
  Verificación: `go test ./internal/cli -run 'SDDArchiveCheck|ArchivedTrace'`
- [ ] 6.7 RED y GREEN: en `cmd/axiom/main.go` y `cmd/axiom/main_test.go`, registro del subcomando, texto de ayuda (`:171` y `:2211`) y lista de subcomandos (`main_test.go:52`).
  Cubre: REQ-47.8
  Verificación: `go test ./cmd/axiom -run 'Help'`
- [ ] 6.8 RED (G1): en `internal/sddstatus` y `internal/kickoff`, con un cambio `checkpointed` y las puertas aprobadas, tras ejecutar el verbo (que reescribe tareas, deltas y `verify-report.md`) las puertas siguen `approved` y `nextRecommended` no cambia; en `kickoff/machine.go` (`resolveGateStatus`) una aprobación es terminal con independencia del digest; `continuous` no evalúa puertas.
  Cubre: REQ-47.8, REQ-21.17
  Verificación: `go test ./internal/sddstatus -run 'Gate'` y `go test ./internal/kickoff/... -run 'Gate|Digest'`
- [ ] 6.9 GREEN (G1): si 6.8 lo exige, modificar `internal/kickoff/machine.go` para que la aprobación sea terminal con independencia del digest; si ya lo es, solo se conserva la prueba de regresión y se anota en el commit.
  Cubre: REQ-47.8
  Verificación: `go test ./internal/kickoff/... ./internal/sddstatus -run 'Gate|Digest'`
- [ ] 6.10 Cierre de U6: `go vet ./internal/livespec/... ./internal/cli ./cmd/axiom ./internal/sddstatus/... ./internal/kickoff/...` y commit `feat(sdd): verbo archive-integrate que asigna IDs y compone todo el cambio`.
  Cubre: REQ-47.8, REQ-21.17, REQ-47.5, REQ-47.6
  Verificación: `go test ./internal/livespec/... ./internal/kickoff/...` y `go test ./internal/cli -run 'SDDArchive'`

## Fase 7 (U7): `status.requirements`

Depende de: U2. Paquetes: `internal/sddstatus`, contrato compartido.

- [ ] 7.1 RED: en `internal/sddstatus/requirements_test.go`, `requirements.modifies` y `requirements.creates` ordenados; sin `id` para nombres sin ID (resolución por título, sin leer la canónica); `provisional: true` con `id` solo si el nombre lleva ID válido o provisional; RENAMED usa el nombre antiguo; capacidad nueva como spec completa cuenta como ADDED; el campo se omite byte a byte sin deltas legibles (incluido almacén `engram` o `none`), con cambio archivado (`status.Archived != nil`) y con patrón inaccesible; `nextRecommended`, `dependencies` y motivos iguales con y sin el campo.
  Cubre: REQ-47.9
  Verificación: `go test ./internal/sddstatus -run 'Requirements'`
- [ ] 7.2 GREEN: crear `internal/sddstatus/requirements.go` con `decorateRequirements` y `RequirementsProjection`; modificar `status.go` (campo `Requirements` con `omitempty`, llamada tras `decorateISO`) y `status_v2.go`.
  Cubre: REQ-47.9
  Verificación: `go test ./internal/sddstatus -run 'Requirements|Status'`
- [ ] 7.3 RED y GREEN: añadir la sección «Requirements» a `internal/assets/skills/_shared/sdd-status-contract.md` y su pin en `internal/assets/assets_test.go` (campos, ausencia de `id`, los tres casos de omisión).
  Cubre: REQ-47.9
  Verificación: `go test ./internal/assets -run 'StatusContract'`
- [ ] 7.4 Cierre de U7: `go vet ./internal/sddstatus/...` y commit `feat(sddstatus): proyectar requirements.modifies y requirements.creates en status`.
  Cubre: REQ-47.9
  Verificación: `go test ./internal/sddstatus -run 'Requirements'` y `go test ./internal/assets -run 'StatusContract'`

## Fase 8 (U8): Guías, pins y documentación

Depende de: U5, U6, U7. Paquetes: skills, `agentguidance`, `workspace`, docs.

- [ ] 8.1 RED: pins nuevos de REQ-47.10 en `internal/assets/sdd_iso_pins_test.go` y `internal/assets/assets_test.go`: `sdd-spec` describe el provisional (ADDED, `Cubre:`, forma con ángulos para textos validados); `sdd-iso-convention` cubre provisional en ADDED, `Cubre:` y matriz y nombra el verbo; `sdd-archive` invoca una sola vez `axiom sdd archive-integrate ... --json` tras `archive-check`, manda `--check` tras cada rebase de la rama de archivo y no conserva el bucle `archive-compose` por dominio ni el bloque `cp` de capacidad nueva.
  Cubre: REQ-47.10, REQ-47.8
  Verificación: `go test ./internal/assets -run 'SDDArchive|SDDSpec|ISO'`
- [ ] 8.2 GREEN: modificar `internal/assets/skills/sdd-archive/SKILL.md` (paso único Go entre `archive-check` y `archive sync`, y con contexto mantenido entre `archive context prepare` y las ediciones; tabla de `assignments` en `archive-report.md`; instantánea y `mv` del paso 3 después), `internal/assets/skills/sdd-spec/SKILL.md` (provisional en la sección ADDED y línea ISO), `internal/assets/skills/sdd-spec/references/openspec-adoption.md` (paso 7, conservando la cadena fijada de `assets_test.go:3686`) y `internal/assets/skills/_shared/sdd-iso-convention.md`. Verificar sin cambiar `internal/assets/skills/sdd-archive/references/technical-context-integration.md` (su pin «Run the default Step 2 blocks…» sigue cierto).
  Cubre: REQ-47.10, REQ-47.8, REQ-21.17
  Verificación: `go test ./internal/assets -run 'SDDArchive|SDDSpec|ISO'`
- [ ] 8.3 Actualizar al flujo nuevo los pins existentes `TestSDDArchiveFollowsTheArchiveBranchFlow`, `TestSDDArchiveStoreSpecificFilesystemContract`, `TestSDDArchiveTechnicalContextBranch` y `TestSDDArchiveRequiresArchiveCheck` en `internal/assets/assets_test.go`.
  Cubre: REQ-47.10, REQ-21.17
  Verificación: `go test ./internal/assets -run 'TestSDDArchive'`
- [ ] 8.4 RED: en `internal/components/agentguidance/routing_test.go`, el paso 7 (regla 3 de REQ-24.58) dice que el ADDED lleva un provisional y que `axiom archive integrate --odd <feature>` asigna el número y reescribe el documento, en un solo párrafo, y el comando aparece una sola vez (`:805`).
  Cubre: REQ-24.58
  Verificación: `go test ./internal/components/agentguidance -run 'Routing'`
- [ ] 8.5 GREEN: modificar `internal/components/agentguidance/routing.go` (paso 7, `:77`) con ese texto.
  Cubre: REQ-24.58
  Verificación: `go test ./internal/components/agentguidance -run 'Routing'`
- [ ] 8.6 Manual: modificar `docs/manual-de-inicio.md` en `:657` y `:662` (la integración ODD ya no «no modifica el documento ODD»: ADDED con provisional, reescritura del documento, `--check`) y en `:431` (verbo de archivado); conservar las cadenas fijadas por `internal/workspace/docs_pin_test.go` y añadir los pins nuevos que hagan falta.
  Cubre: REQ-47.10, REQ-30.2
  Verificación: `go test ./internal/workspace -run 'DocsPin'`
- [ ] 8.7 Goldens: comprobar con los tests de goldens ya existentes que ninguno de los 63 contiene `Spec viva` ni `archive integrate --odd`; regenerar solo si una prueba de goldens falla. Anotar el resultado en el commit.
  Cubre: REQ-47.10
  Verificación: `go test ./internal/components/agentguidance -run 'Golden'`
- [ ] 8.8 Cierre de U8: `go vet` de los paquetes tocados y commit `docs(sdd): guías del provisional, paso de archivado Go y paso 7 ODD con pins`.
  Cubre: REQ-47.10, REQ-24.58, REQ-47.8
  Verificación: `go test ./internal/assets -run 'SDDArchive|SDDSpec|ISO|StatusContract'` y `go test ./internal/components/agentguidance -run 'Routing'` y `go test ./internal/workspace -run 'DocsPin'`

## Fase 9: Verificación final localizada

Depende de: U1 a U8. Sin commit propio salvo corrección de lo que falle.

- [ ] 9.1 Ejecutar, paquete a paquete y sin la batería completa: `go test ./internal/reqid/... ./internal/openspecdoc/... ./internal/isoevidence/... ./internal/livespec/... ./internal/kickoff/...` y `go vet` de los mismos; anotar cada resultado observado en `verify-report.md`.
  Cubre: REQ-47.1, REQ-47.2, REQ-47.3, REQ-47.4, REQ-47.5, REQ-47.6
  Verificación: `go test ./internal/reqid/... ./internal/openspecdoc/... ./internal/isoevidence/... ./internal/livespec/... ./internal/kickoff/...`
- [ ] 9.2 Ejecutar los paquetes grandes solo con `-run`: `go test ./internal/cli -run 'ArchiveIntegrate|SDDArchive|SDDTrace'`, `go test ./internal/sddstatus -run 'Requirements|Gate|ISO'`, `go test ./internal/assets -run 'SDDArchive|SDDSpec|ISO|StatusContract'`, `go test ./internal/components/agentguidance -run 'Routing'`, `go test ./internal/workspace -run 'DocsPin'` y `go test ./cmd/axiom -run 'Help'`.
  Cubre: REQ-47.7, REQ-47.8, REQ-47.9, REQ-47.10, REQ-21.17, REQ-30.2, REQ-30.3, REQ-30.4, REQ-30.22, REQ-29.3, REQ-29.4, REQ-24.58
  Verificación: los comandos citados en esta misma tarea
- [ ] 9.3 Nota para el archive (no es edición de esta fase): corregir a mano el `## Purpose` obsoleto de `openspec/specs/requirement-identifiers/spec.md`, que dice que la capacidad no asigna provisionales; REQ-44.3 no se modifica. La suite completa (`go test ./...`) la pasa la CI, no se ejecuta en local.
  Cubre: REQ-47.1, REQ-47.2
  Verificación: lectura del `## Purpose` tras el archive; la CI ejecuta la batería entera

## Orden y paralelismo

```
U1 ──► U2 ──► U7 ──────────────┐
  └──► U3 ──┐                   │
U2 ─────────┴─► U4 ──► U5 ──────┼──► U8 ──► Fase 9
                  └──► U6 ──────┘
```

- Secuencial por dependencia: U1 → (U2, U3) → U4 → (U5, U6) → U8.
- Paralelizables entre sí, por superficies de edición disjuntas: U2 y U3; U5 y U6 (comparten `livespec/report.go` solo de lectura; si hay conflicto, U5 va antes); U7 puede ir tras U2 y en paralelo a U3 a U6.
- Con un solo escritor (lo previsto), el orden de commits es U1, U2, U3, U4, U5, U6, U7, U8.
