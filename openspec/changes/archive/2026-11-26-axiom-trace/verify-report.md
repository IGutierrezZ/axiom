# Informe de verify: axiom-trace (C-03-d), rol fullstack

Alcance: `git diff ee1bbf03..HEAD` (commits bfc94c78..8bf5366e, U1..U10) frente a `specs/requirement-trace/spec.md` (prov:trace:1..15), `design.md` y `tasks.md`. `strict_tdd: false`.

## Comprobaciones ejecutadas (solo localizadas)

| Comando | Resultado |
|---|---|
| `go test ./internal/reqtrace -count=1` | ok |
| `go test ./internal/cli -run 'AxiomTrace\|SDDTrace\|TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign' -count=1` | ok (73,9 s) |
| `go test ./internal/reqid -run 'TestNoRequirementIDFormatOutsideReqid\|TestProvisionalLiteralOnlyInReqid' -count=1` | ok |
| `go test ./cmd/axiom -run 'TraceVerb\|Help' -count=1` | ok |
| `go test ./internal/workspace -run DocsPin -count=1` | ok |
| `go vet ./internal/reqtrace ./internal/cli ./cmd/axiom` | sin hallazgos |
| `gofmt -l internal/reqtrace internal/cli cmd/axiom` | sin ficheros |
| `git diff ee1bbf03..HEAD -- internal/cli/sdd_trace.go` | vacío: no cambió |

No se ejecutó la suite completa ni se compilaron binarios (la batería completa queda para CI).

## Cumplimiento técnico

- Código muerto: todos los símbolos de producción de `internal/reqtrace` son alcanzables desde `cmd/axiom` (`main.go` `case "trace"` -> `runTrace` -> `cli.RunTrace` -> `reqtrace.ParseID`/`Collect` -> escaneo, clasificación, equivalentes, anclaje y orden). Las constantes `Warn*`, `Anchor*`, `NoAliases`, `RefError` e `IDError` tienen uso en producción. Comprobado por lectura y recuento de usos; no se ejecutó el trinquete de CI.
- `sdd_trace.go` sin cambios.
- Solo lectura: git únicamente vía `gitcommon.Run` con la raíz de cada fuente; la prueba `TestAxiomTraceIsReadOnlyAndUsesTheProjectRepository` cubre prov:trace:10.
- Ayuda global y propia, manual de inicio y `docs_pin_test` actualizados.

## Escenarios de la spec frente a pruebas

- trace:1 ayuda y falta de ID: `TestTraceVerbIsListedInTheGlobalHelp`, `TestTraceVerbWithoutIDFailsWithUsage`, `TestTraceVerbHelp`, `TestAxiomTraceHelp`, `TestAxiomTraceRejectsBadArgumentsWithoutGit`.
- trace:2 patrón: `TestRequirementTraceParseID`, `TestAxiomTraceInvalidIDNamesPatternAndContinuationWithoutGit`, `TestAxiomTraceProjectPatternDecidesTheID`.
- trace:3 y 4 equivalentes y alias: `TestRequirementTraceEquivalents*` (provisional/definitivo, Previous, alias inyectado, ID sin traza), `TestRequirementTraceCollectProvisionalAndDefinitiveGiveSameResult`.
- trace:5 y 6 SDD: `TestRequirementTraceSDD*` (roles por sección, Cubre con tareas, activo ignorado, misma carpeta/nombre, bug con ambas secciones, increment).
- trace:7 ODD: `TestRequirementTraceODD*` (roles/tipo/Cubre, traza, antiguo sin traza, no parsea con advertencia, no integrado en `TestRequirementTraceAnchorODD`).
- trace:8 anclaje: `TestAxiomTraceSDDAndODDIntegratedWithMergeSquashAndRebase`, `TestRequirementTraceAnchor*`, found:false.
- trace:9 y 11 JSON y vacío: `TestRequirementTraceCollectEndToEnd`, `...ValidIDWithoutAppearancesHasEmptyLists`, `TestRequirementTraceResultNormalizeEmitsEmptyLists`, `TestAxiomTraceRenderText`.
- trace:12 rama: `TestAxiomTraceMissingBranchNamesBranchAndRemedyWithoutFetch`, `TestAxiomTraceBranchExplicitMissingAndDeterministicOrder`.
- trace:13 orden: `TestRequirementTraceSort*`, `TestRequirementTraceCollectIsDeterministic`.
- trace:14 multirepo: `TestAxiomTraceMultirepo*`, `TestRequirementTraceAnchorUsesEachSourcesOwnHistory`.
- trace:15 de punta a punta: `TestAxiomTraceSDDAndODDIntegrated...` y `TestRequirementTraceCollectEndToEnd` (incluye el bug SDD `covered`).

## Desviaciones declaradas por el apply

- `Source.Repo` añadido: campo interno necesario para trace:14 (anclar cada ruta con su raíz git); no contradice la spec. Aceptable.
- `odd_unparseable` solo si el documento cita el ID: la spec (trace:7) lo condiciona en el escenario a que cite el ID; la lectura literal del requisito lo exigiría para cualquier documento ilegible. Evita ruido en documentos ajenos a la consulta. Aceptable, con WARNING menor de redacción.
- `odd_repository_unavailable` siempre que `odd/tasks` exista sin git: coherente con trace:14 (advertencia que nombra el repositorio, sin commit ajeno). Aceptable.
- ID admitido entre opciones: la spec solo exige posicional obligatorio; sin conflicto. Aceptable.

## Tareas

Estado observado en `tasks.md` según el apply (U1..U10); no se reescribe.

## Hallazgos

- CRITICAL: 0.
- WARNING: 1. Aclarar en el diseño o la spec que `odd_unparseable` solo se emite si el documento cita el ID (trace:7 lo dice sin condición en el requisito).
- SUGGESTION: 1. Ejecutar el trinquete de código muerto en CI para confirmar la alcanzabilidad comprobada por lectura.

**Estado:** limpio
