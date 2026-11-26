# Propuesta: HTML explicativo de spec y design (`resumen-spec.html` y `resumen-design.html`)

Cambio C-06-b del plan maestro (`docs/plan/backlog-2026-11/06-ui-human-loop.md`, líneas 41-117).

## Intención

Los `spec.md` y `design.md` de un cambio SDD son densos para quien no es técnico, y esa densidad favorece que una persona apruebe una puerta sin leer lo que aprueba. El usuario pidió (2026-11-13) que las fases `sdd-spec` y `sdd-design` entreguen, además del md, un HTML explicativo con gráficos y diagramas que entienda cualquiera, visible desde la UI. El HTML lo genera el modelo dentro de la propia fase; no es un derivado determinista. Es exclusivo de SDD y se decide una sola vez al iniciar el cambio, para no gastar tokens en cambios pequeños.

Éxito: con la opción activada, cada cambio tiene un `resumen-spec.html` y un `resumen-design.html` autocontenidos, cuya frescura respecto del md se ve en `axiom sdd status` y en el dashboard, sin alterar nunca qué se aprueba (el md) ni el digest de ninguna puerta.

## Alcance

### Dentro del alcance

- **Declaración de proyecto.** Clave `sdd.explainer_html: ask|always|never` dentro del bloque `sdd:` de `axiom.yaml`, por defecto `ask`, validada por el decodificador estricto del bloque (`workspace validate` rechaza cualquier otro valor). Documentada en `axiom.example.yaml`.
- **Campo sellado.** `explainer_html: yes|no` en `kickoff.yaml` (`omitempty`; ausente equivale a `no`, compatible con cambios previos y con el retrosellado `--infer`, que no lo escribe ni lo hereda). Bandera `axiom sdd kickoff seal --explainer-html yes|no`, rechazada con `--infer`. Resolución: `always` sella `yes` y `never` sella `no` sin preguntar; una bandera que contradiga `always`/`never` se rechaza sin escribir nada; con `ask` y sin bandera se sella `no` con aviso. Independiente del estilo de ejecución.
- **Interacción propia.** Una pregunta única con marcador `Axiom SDD kickoff explainer:`, fuera del prefijo `Axiom SDD preflight ` (el hook y el plugin de OpenCode la ignoran, como la preferencia de RF-PRE-06), formulada una sola vez al iniciar el cambio, después de `axiom branch ensure` y antes de `axiom sdd kickoff seal`, solo con `ask`. Nunca en ODD ni con `always`/`never`. Doctrina en la sección `SDD Change Kickoff and Block Gates` de `internal/assets/skills/_shared/sdd-orchestrator-sections.md`.
- **Proyección en el estado.** `axiom sdd status` proyecta el valor de proyecto (para que el orquestador sepa si preguntar) y el valor sellado en `governance.kickoff` (para que las fases sepan si generar). Documentado en `sdd-status-contract.md`.
- **Generación en las fases.** `sdd-spec` y `sdd-design`, cuando el kickoff sella `yes`, escriben tras el md `resumen-spec.html` o `resumen-design.html` en la raíz de la carpeta del cambio: en castellano, para público no técnico (qué cambia, para quién, flujos antes/después, SVG en línea, glosario), autocontenidos (HTML, CSS y SVG en línea; sin `<script>` ni recursos externos), con `<meta name="axiom-source-digest">` igual a `kickoff.ArtifactDigest` de los md de origen, citando el ID de requisito o decisión de cada bloque, sin añadir nada que no esté en el md y diciendo expresamente que lo que se aprueba es el md. Si una puerta se rechaza, la fase regenera md y HTML. Actualización de los SKILL.md y de las copias por agente existentes (`claude`, `cursor`, `copilot`).
- **Frescura informativa.** Estado `fresh|stale|missing` por `spec` y `design` en `axiom sdd status` y en el dashboard, calculado con `sddstatus.ChangeArtifactPaths` (sin una tercera copia de la regla de descubrimiento de specs) y también en `continuous`. Nunca cambia `nextRecommended`, las dependencias ni el digest de ninguna puerta.
- **Fuera del digest de las puertas.** Test que fija que crear o modificar un `resumen-*.html` no cambia el digest de `spec` ni de `design`.
- **Dashboard.** Endpoint de solo lectura que sirve únicamente `resumen-spec.html` y `resumen-design.html` de un cambio, por lista blanca, rechazando `..` y separadores en el nombre del cambio, con `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; img-src data:` y `X-Content-Type-Options: nosniff` fijadas en el propio handler (no en `withHeaders`). Pestaña «Explicación» en el detalle del incremento con `iframe sandbox` (sin `allow-same-origin` ni `allow-scripts`) y aviso de desactualizado.
- **Comprobación automática** en tests de assets de que la doctrina de las skills exige las restricciones del HTML, y test gemelo del de RF-PRE-06 que fija que el hook ignora la pregunta nueva.

### Fuera del alcance (del plan)

- ODD.
- Generar HTML para proposal, tasks o verify.
- Diagramas con librerías JS (mermaid, markmap).
- Editar el HTML desde la UI.
- Que el HTML entre en el digest de una puerta.
- Añadidos de esta propuesta: un verbo que valide en ejecución el HTML producido por el modelo; corregir `assets/js/core/spec.js:25` (`REQ_ID` no reconoce `RF-*`, preexistente); retirar `Access-Control-Allow-Origin: *` y la CSP global (son de C-07-h).

## Decisiones tomadas por el coordinador (con motivo)

1. **Clave dentro de `sdd:` y no de primer nivel** (desvío del plan): las claves de primer nivel no registradas se ignoran sin error (`internal/workspace/declarations.go:72`), así que `explainer_html` quedaría sin validar; el bloque `sdd:` agrupa lo exclusivo de SDD y ya tiene decodificador estricto.
2. **Ausente equivale a `no`, y `ask` sin bandera sella `no` con aviso**: no rompe cambios ya sellados, `--infer` ni llamadas existentes a `seal` (dashboard incluido); la doctrina obliga a preguntar antes de sellar.
3. **Interacción propia, no cuarto grupo del preflight**: P10 (README, fila P10) sacó la estrategia de PR del preflight y dejó el multirepo sin preguntas; un marcador distinto evita tocar hook, plugin y topología, con precedente exacto en RF-PRE-06.
4. **HTML en la raíz del cambio**: uno por cambio, no por dominio; se publica con `axiom sdd publish` junto al resto de la documentación del cambio sin cambios en la publicación.
5. **Frescura solo informativa**: la puerta aprueba el md, que es el contrato.
6. **CSP en el handler**: testeable con `httptest.NewServer(server.Router())` y sin choque con C-07-h, que reescribirá `withHeaders`.

## Capacidades

### Capacidades nuevas

Ninguna.

### Capacidades modificadas

- `sdd-lane-kickoff`: declaración `sdd.explainer_html` como entrada del sellado, campo sellado `explainer_html`, bandera `--explainer-html`, reglas de resolución y rechazo, compatibilidad con ausencia y `--infer`, y proyección en `governance.kickoff` (requisitos nuevos, `prov:KOF:n`).
- `sdd-preflight-configuration`: la pregunta del HTML explicativo es una interacción propia de marcador `Axiom SDD kickoff explainer:`, una vez por cambio, solo con `ask`, nunca en ODD, ignorada por hook y plugin (requisito nuevo, `prov:PRE:n`).
- `sdd-block-review-gates`: contrato del HTML que generan `sdd-spec` y `sdd-design` (contenido, autocontención, digest de origen, cita de IDs, el md es lo que se aprueba, regeneración tras rechazo), exclusión del digest de las puertas y frescura `fresh|stale|missing` informativa en el estado (requisitos nuevos, `prov:GAT:n`).
- `local-web-dashboard`: endpoint de los dos ficheros con CSP, lista blanca y rechazo de traversal, y pestaña «Explicación» con `iframe sandbox` y aviso de desactualizado (requisitos nuevos, `prov:WEB:n`).
- `sdd-iso-evidence`: RF-ISO-01 (MODIFIED) amplía la lista cerrada de claves del bloque `sdd:` con `explainer_html` y sus valores admitidos. Es donde vive hoy la lista cerrada del bloque (`iso`, `id_pattern`); `workspace-topology` no especifica el bloque `sdd:` más allá de mencionarlo en RF-TOP-17, por lo que no se modifica.

## Enfoque

Enfoque 1 de la exploración: valor de proyecto en `sdd:`, instantánea por cambio en `kickoff.yaml` siguiendo el patrón de `DocsPublication` (tipo en `internal/kickoff/types.go`, mapa cerrado en `schema.go`, bandera en `args.go`, salida en `internal/cli/sdd_kickoff.go`), e interacción propia como RF-PRE-06. La frescura se implementa en un fichero nuevo `internal/sddstatus/explainer.go` como decorador aditivo (patrón de `decorateRequirements`), comparando el `<meta>` con `kickoff.ArtifactDigest` sobre `ChangeArtifactPaths`; el dashboard lo reutiliza en `IncrementDetailDTO`. La doctrina va en un párrafo propio al final de la sección del kickoff, para reducir el solape con C-05-d. Los goldens afectados se regeneran.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/workspace/sdd.go`, `axiom.example.yaml`, `internal/workspace/example_config_test.go` | Modificado | Clave `explainer_html` en `SDDConfig` y plantilla |
| `internal/kickoff/{types,schema,args}.go` | Modificado | Campo sellado, validación y bandera |
| `internal/cli/sdd_kickoff.go` | Modificado | Resolución desde `axiom.yaml`, aviso, ayuda y JSON |
| `internal/sddstatus/explainer.go` (nuevo), `status.go`, `status_v2.go` | Nuevo/Modificado | Proyección del valor y frescura |
| `internal/cli/sdd_preflight_hook_test.go` | Modificado | Test de que el hook ignora la pregunta |
| `internal/cli/sdd_gate_digest.go` (solo tests) | Modificado | Test de exclusión del HTML del digest |
| `internal/dashboard/{server,service,types}.go`, `assets/index.html`, `assets/js/core/router.js`, `assets/js/screens/detalle.js` | Modificado | Endpoint, DTO y pestaña |
| `internal/assets/skills/_shared/{sdd-orchestrator-sections,sdd-status-contract}.md` | Modificado | Doctrina de la pregunta y contrato del estado |
| `internal/assets/skills/sdd-{spec,design}/SKILL.md`, `internal/assets/{claude,cursor}/agents/sdd-{spec,design}.md`, `internal/assets/copilot/agents/sdd-{spec,design}.agent.md` | Modificado | Generación del HTML |
| `testdata/golden/` (11 de orquestador + 2 de agentes) | Modificado | Regeneración |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| C-05-d en paralelo toca la misma sección doctrinal y los mismos 11 goldens | Alta | Párrafo en bloque propio al final de la sección; el segundo en fusionarse rebasa y regenera goldens |
| Coste en tokens en `sdd-spec` y `sdd-design` | Media | Defecto `ask`, una sola pregunta por cambio; `never` lo apaga para el proyecto |
| El HTML simplifica de más o afirma algo que no está en el md | Media | Cita obligatoria de IDs, aviso de que se aprueba el md y revisión junto al md en la parada |
| El modelo incumple la autocontención (script, URL externa) | Media | CSP estricta e `iframe sandbox` neutralizan el contenido; test de assets sobre la doctrina; sin validación en ejecución (fuera de alcance) |
| Path traversal en el endpoint (`locateIncrement` no valida `..`) | Media | Validación propia del nombre y lista blanca de ficheros; tests con `..`, `%2e%2e` y separadores |
| Una llamada con `ask` sin bandera sella `no` sin que se haya preguntado | Baja | Aviso en la salida de `seal` y doctrina que exige preguntar antes |
| Choque con C-07-h en cabeceras | Baja | CSP solo en el handler; `withHeaders` no se toca |

## Plan de reversión

Revertir el commit o la PR del cambio. El campo es `omitempty` y su ausencia equivale a `no`; el paquete `internal/kickoff` decodifica sin `KnownFields`, así que el binario anterior ignora la clave `explainer_html` de los `kickoff.yaml` ya sellados (a confirmar en diseño con un test de lectura heredada, como el de `internal/kickoff/ledger_test.go:185`); como último recurso basta borrar esa línea. Retirar `sdd.explainer_html` de `axiom.yaml` devuelve la validación al estado previo. Los `resumen-*.html` ya generados son ficheros inertes que pueden borrarse sin efecto en puertas ni estado.

## Dependencias

- C-06-a (detalle del dashboard con `readArtifactFiles`): fusionado.
- Hook del preflight por marcador (sección 01): fusionado; la pregunta nueva no depende de él al usar un prefijo distinto.
- C-05-b (publicación `docs_publication: auto`): dependencia blanda; el HTML se publica con el mecanismo existente.

## Criterio de terminado

- [ ] Con `sdd.explainer_html: ask`, la pregunta aparece una sola vez al iniciar el cambio y su respuesta queda en `kickoff.yaml`.
- [ ] Con `always` o `never` no se pregunta y se sella `yes` o `no`; una bandera contradictoria se rechaza.
- [ ] En ODD la pregunta nunca aparece.
- [ ] Un valor no admitido en `sdd.explainer_html` falla en `axiom workspace validate` y al sellar.
- [ ] Con `yes`, `sdd-spec` y `sdd-design` escriben su HTML con `axiom-source-digest` y sin `<script>` ni URLs externas (comprobación automática).
- [ ] Si se modifica el md, `axiom sdd status` y la UI marcan el HTML como desactualizado; sin HTML, como ausente; nada de esto cambia `nextRecommended`.
- [ ] El digest de las puertas `spec` y `design` no cambia al crear o modificar el HTML (test).
- [ ] La UI sirve el HTML con CSP y `sandbox`; un test comprueba las cabeceras y que el endpoint rechaza cualquier otro fichero o una ruta con `..`.
- [ ] El hook del preflight ignora la pregunta `Axiom SDD kickoff explainer:` (test).
