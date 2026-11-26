# Informe de verificación — html-explicativo (C-06-b)

Rol único `fullstack`. Alcance: `git diff 98a2af597..HEAD` (rama `inc/html-explicativo`, commits `0bf785d3`..`562ae2dd`). TDD desactivado.

## Tareas

12/12 marcadas `[x]` en `tasks.md`; no hay tareas pendientes.

## Cumplimiento técnico

Comandos ejecutados (todos con salida 0; solo `-run` localizados, sin binarios sueltos):

| Paquete | Filtro | Resultado |
|---|---|---|
| internal/explainer | (todo el paquete) | ok |
| internal/workspace | `TestDecodeSDD\|TestSDDLoad\|TestExample\|TestDeclarationsExplainer` | ok |
| internal/kickoff | `Explainer\|TestParseSealArgs\|TestInfer\|TestLegacy` | ok |
| internal/cli | `TestRunSDDKickoffSeal.*Explainer\|TestGateArtifactDigest.*Explainer\|TestSDDPreflightHookIgnores\|SDDKickoff\|SDDStatus` | ok |
| internal/sddstatus | `Explainer\|TestSDDStatusV2CleanBreak\|TestExplainerISO` | ok |
| internal/sddpublish | `Explainer` | ok |
| internal/dashboard | `Explainer\|TestIncrementDetail\|TestRouterDefinesEveryRoute\|TestExplainerTab` | ok |
| internal/assets | `Explainer\|TestOrchestratorSectionDriftDoesNotGrow\|TestOpenCode\|TestSDDISOConvention\|TestSkill\|TestShared` | ok |
| internal/components | `TestGolden` | ok |
| `gofmt -l` (paquetes tocados) | — | sin salida |
| `go vet` (paquetes tocados) | — | sin avisos |

## Matriz requisito → evidencia

- local-web-dashboard WEB:1 (endpoint): `handleExplainer` en `server.go` (lista blanca de dos nombres vía `explainer.ValidKind`, regex de nombre de cambio que excluye `..`, separadores, NUL y codificados ya decodificados; 400 antes de leer disco; 404 por `ErrExplainerNotFound`; 405 salvo GET/HEAD; CSP exacta, nosniff, Content-Type). Pruebas: `TestExplainerEndpointServesWhitelistedHTML`, `...RejectsBadParameters`, `...RefusesUnsafeFiles`.
- WEB:2 (pestaña): `detalle-explicacion.js` (iframe `sandbox=""`, aviso `stale`, estados ausente/no activado), `service.go incrementExplainer` con la misma función de frescura que el status. Pruebas: `TestIncrementDetailExplainer*`, `TestExplainerTab`.
- sdd-block-review-gates GAT:1..4 (contrato, fidelidad, digest, frescura): `internal/explainer`, `sddstatus/explainer.go`, skills `sdd-spec`/`sdd-design`; pruebas `TestGateArtifactDigest.*Explainer`, `sddstatus/explainer_test.go` (fresh/stale/missing, continuous, ISO sin bloqueo).
- sdd-lane-kickoff KOF:1..4 y sdd-iso-evidence (clave `explainer_html`): `workspace/sdd.go`, `kickoff/{args,explainer,schema,types}.go`, `sdd_kickoff.go`; pruebas de workspace, kickoff y `TestRunSDDKickoffSeal.*Explainer`.
- sdd-preflight-configuration PRE:1: orquestador, hook y plugin OpenCode; `TestSDDPreflightHookIgnores*`, `TestOpenCode`, goldens.

## Conformidad con design.md y criterio de terminado (plan 06, líneas 104-111)

Cumplidos todos: pregunta única solo con `ask`; `always`/`never` sin pregunta; ODD nunca; `axiom-source-digest` y ausencia de `<script>` y URLs externas validadas por `explainer.Validate`; frescura en status y UI; digest de puertas intacto (prueba); CSP en el handler (decisión 8) más `sandbox` en el iframe, con pruebas de cabeceras y de rechazo de `..`.

## Seguridad del endpoint

Sin traversal (ruta construida por el servidor con carpeta del cambio y nombre de lista blanca; ficheros no regulares rechazados; límite 5 MiB), CSP exacta, solo lectura, `Cache-Control: no-store`.

## Citas REQ-n.m

Sin citas `REQ-n.m` en los `specs/*/spec.md` del cambio.

## Hallazgos

- CRITICAL: 0
- WARNING: 0
- SUGGESTION: 1. La CSP no incluye la directiva `sandbox`; el aislamiento depende del atributo del iframe (la spec fija el valor exacto de la CSP, por lo que es conforme). Limitación: no se ejecutó la suite completa (política local; la cubre CI).

**Estado:** limpio
