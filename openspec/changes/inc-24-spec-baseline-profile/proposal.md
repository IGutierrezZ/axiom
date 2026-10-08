# Propuesta: Baseline del proyecto y raíz multirepo para SDD (inc-24-spec-baseline-profile)

> Fase `propose` · rama `feat/inc-24-spec-baseline-profile` · 2026-10-04 · entrada: `exploration.md` (Engram `sdd/inc-24-spec-baseline-profile/explore`).
> Kickoff sellado: `checkpointed`, `handoff_policy: none`, rol único `fullstack` (`blocking`), `deployment_target: local`.
> Los supuestos de §3 y las preguntas de §4 están **pendientes de confirmación**: nada de esas dos secciones se da por decidido.
> Revisión 1 (2026-10-04): el contexto técnico deja de ser de solo lectura y lo mantiene el flujo SDD (requisito D5, feedback vinculante del usuario).

## 0. Resumen

Axiom no puede gobernar con SDD un proyecto adoptado en la topología multirepo canónica: el motor resuelve siempre `<cwd>/openspec` y archive solo integra en `openspec/specs/` con encabezados `### Requirement:`. inc-24 lo resuelve en un único cambio con tres rebanadas en orden de dependencia A → B → C (decisión D4).

| Rebanada | Entrega | Caso de adopción que habilita (D3) |
|---|---|---|
| **A. Resolver de raíz** | Un único resolver del planning home, válido desde la carpeta maestra, el repo de specs o un repo de código, consumido por todos los verbos SDD y por los consumidores de catálogo | Sin spec: OpenSpec desde cero en multirepo |
| **B. Declaración y guardas** | Bloques `baseline:` y `technical_context:` en `axiom.yaml` (el contexto, con dos modos: mantenido y de referencia), proyección aditiva en el status, guardas contra una segunda baseline y lectura del contexto técnico por las fases | Spec mala o desactualizada: contexto de referencia (solo lectura) y OpenSpec desde cero |
| **C. Integración en la baseline y en el contexto del proyecto** | Compose y archive sobre la baseline declarada (patrón de encabezado, identidad por ID, plantilla de render, duplicados, documentos no-requisito), skills de lectura y mantenimiento del contexto técnico: delta de contexto por cambio (design y apply), aviso de verify e integración en archive | Spec direccionable y fiable; spec fiable en prosa o tablas, con verificación; además, contexto técnico mantenido al día (D5) |

Principio rector (D1): **los cambios son de Axiom; la spec viva y el contexto técnico son del proyecto.** El ciclo `openspec/changes/<change>/` no cambia; la baseline y el contexto técnico del proyecto se quedan en su formato y ubicación, y archive integra en ellos en lugar de en `openspec/specs/`. El trabajo en vuelo del proyecto adoptado termina con su flujo antiguo.

Requisito vinculante D5 (feedback del usuario, 2026-10-04): el contexto técnico no es de solo lectura. Cuando un cambio altera algo que documenta el contexto mantenido, el flujo SDD lo actualiza: el cambio lleva su delta de contexto, simétrico a las delta specs, y archive lo integra. La documentación declarada como de referencia no se modifica nunca.

## 1. Propósito (Intent)

### 1.1 Problema

| Hueco | Evidencia |
|---|---|
| La raíz es el cwd. En la topología canónica (carpeta maestra no versionada con `.axiom-workspace` → `<specs_repository>/axiom.yaml`) ningún verbo SDD encuentra el planning home, aunque `workspace.LoadConfig` ya resuelve el puntero. | `sddstatus.resolveWorkspaceRoot` (`internal/sddstatus/status.go:1166`); `cli.resolveGovernanceChangeRoot` (`internal/cli/sdd_governance.go:31`); `os.Stat(<raíz>/axiom.yaml)` previos en `cmd/axiom/main.go` e `internal/cli/sdd_kickoff.go` |
| Tres convenciones de layout incompatibles para `specs_repository`: motor SDD; livingdoc, dashboard y skillregistry; perfil knowledge. | `exploration.md` §2.4 |
| El compose solo reconoce `### Requirement:` con identidad por texto exacto y, ante nombres repetidos, aplica en silencio sobre la primera coincidencia. | `internal/sddstatus/openspec_archive_compose.go:33,308-315` |
| Seis creadores de `openspec/specs/` u `openspec/INDEX.md` no consultan si el proyecto ya tiene baseline. | `axiom init`, skill `sdd-init`, rama "Main Spec Does NOT Exist" de `sdd-archive`, `knowledge sweep`, grabador de knowledge y `axiom archive coldstart` (`exploration.md` §2.5) |
| El contexto técnico no existe como concepto ni se mantiene: cuando un cambio altera algo que documenta, ninguna fase lo actualiza. | Solo `governance.context` (sin lector Go) y el `context:` de `openspec/config.yaml`; requisito D5 del usuario |

### 1.2 Por qué ahora

La adopción de proyectos con documentación propia (tabla D3) está bloqueada por estos huecos. El coste es acotado: el ciclo de vida del cambio es change-local (estado, kickoff, gates y handoff nunca leen la baseline; `internal/sddstatus/status_test.go:1169-1210`), así que externalizar la baseline solo afecta a la resolución de raíz, a archive y compose, a las skills de lectura, a init y a los consumidores de catálogo.

### 1.3 Resultado esperado

- Desde cualquier directorio del workspace canónico, todos los verbos SDD operan sobre el mismo planning home.
- Un proyecto declara en `axiom.yaml` dónde están su baseline y su contexto técnico. Sin declaración, el comportamiento es idéntico al actual.
- Archive integra el cambio en la baseline declarada de forma determinista y verificable, y nunca crea una segunda baseline.
- El contexto técnico mantenido se actualiza en el mismo flujo cuando un cambio altera algo que documenta, con traza por entrada; el contexto de referencia no se modifica nunca.

## 2. Alcance (Scope)

### 2.1 Dentro de alcance (In Scope)

**Rebanada A: resolver de raíz**

- A1. Primitivo de resolución en `internal/workspace`: desde cualquier directorio de partida devuelve carpeta maestra, raíz del repo de specs, planning home y origen de la resolución.
- A2. Los dos puntos de entrada del motor (`sddstatus.resolveWorkspaceRoot`, `cli.resolveGovernanceChangeRoot`) y los resolvers de `cmd/axiom/main.go` (cambio, handoff, `role barrier`) delegan en él; desaparecen los `os.Stat(<raíz>/axiom.yaml)` que ciegan el puntero.
- A3. livingdoc (`Service` y `Synthesizer`), knowledge (`SpecsRoot`), dashboard y TUI obtienen la raíz del mismo resolver: una sola convención en lugar de tres.
- A4. `monorepo-embedded` resuelve exactamente a `<raíz>/openspec`, sin cambio de comportamiento (R1).
- A5. Rechazo explícito, con continuación nombrada, de la ambigüedad "carpeta maestra con `openspec/` propio y puntero".
- A6. Regla para ejecutores en `_shared/sdd-phase-common.md`: las rutas `openspec/...` se resuelven contra el `workspaceRoot` del status, no contra el cwd.
- A7. Documentación de la receta multirepo canónica en `docs/manual-de-inicio.md`.

**Rebanada B: declaración, guardas y contexto técnico**

- B1. Bloques opcionales `baseline:` y `technical_context:` en `axiom.yaml`, tipados en `WorkspaceConfig` y validados por el loader y por `axiom workspace validate`. Sin bloques, la baseline es OpenSpec (`openspec/specs/`). `technical_context:` distingue dos modos: **mantenido** (el contexto técnico vivo del proyecto, que el flujo SDD mantiene al día; p. ej. `context/` en el proyecto de referencia) y **de referencia** (documentación que solo se usa como entrada; nunca se modifica). La forma YAML exacta se fija en spec y design (Q5, Q10).
- B2. Proyección de ambos localizadores en el status v2 como campos aditivos `omitempty`.
- B3. Guardas contra una segunda baseline en todos los creadores cuando la baseline declarada es del proyecto (alcance exacto en Q11).
- B4. Las fases lectoras (Q10) reciben el contexto técnico declarado de ambos modos como entrada. Ninguna fase escribe en él directamente: los cambios al contexto mantenido viajan en el delta de contexto del cambio (C6) y solo archive los integra (C4).
- B5. Caso "spec mala o desactualizada": esa documentación se declara como contexto de referencia, que nunca se modifica, y OpenSpec queda como baseline por defecto (forma en Q5 y Q10).

**Rebanada C: integración en la baseline y en el contexto del proyecto**

- C1. Compose con perfil de encabezado opcional: patrón con grupos `id` y `title`, nivel, plantilla de render e identidad (`id` o `text`). Sin perfil, comportamiento nativo byte a byte.
- C2. Identidad por ID con rechazo de operaciones ambiguas sobre IDs duplicados (Q7), frontera de bloque por nivel de encabezado, preservación de finales de línea y BOM, y validación de todas las operaciones antes de escribir.
- C3. `axiom sdd archive-compose` resuelve documento y perfil desde `axiom.yaml` con el mismo resolver (`--cwd`, `--document <dominio>`), sin rutas plantilladas en la skill.
- C4. `sdd-archive` integra en la baseline declarada, incluidos los documentos no-requisito según el modo de Q8 y la coexistencia de Q9, e integra el delta de contexto del cambio en las rutas mantenidas con el mismo mecanismo E2: edición dirigida por agente, verificación nativa de presencia y alcance por diff y traza por entrada en la carpeta del cambio y en el informe de archive. Nunca toca las rutas de referencia.
- C5. `sdd-explore`, `sdd-propose`, `sdd-spec` y la convención compartida leen la baseline a través del localizador del status.
- C6. Delta de contexto por cambio, simétrico a las delta specs: un artefacto del cambio (nombre y ubicación en design; p. ej. `context-impact.md` o una sección de `design.md`) que lista qué documentos o secciones mantenidos cambian y por qué, o un "sin impacto" explícito registrado como decisión; su ausencia nunca pasa en silencio. `sdd-design` redacta el impacto previsto y la compuerta `design` lo revisa; `sdd-apply` lo mantiene al día con lo implementado y la compuerta `role-apply` revisa el impacto final.
- C7. `sdd-verify` emite un aviso no bloqueante cuando el diff toca áreas que el contexto documenta (p. ej. manifiestos de dependencias, módulos nuevos de primer nivel, comandos de build o CI) y el delta dice "sin impacto". Es heurístico e informativo: nunca bloquea archive.
- C8. Prerrequisito: spike de solo lectura sobre la baseline del proyecto de referencia (§7).

### 2.2 Fuera de alcance (Out of Scope)

- Correcciones del cold-start de knowledge (`####` del grabador frente a `###` del compose, dominios falsos del sweep, sobrescritura del coldstart), salvo las guardas mínimas de Q11.
- `git init` del review en raíces no git.
- Nombre `gentle-ai` en las continuaciones del review (main ya lo resolvió en #96–#99).
- Rutas `--cwd` con espacios sin comillas.
- Comando `axiom adopt`.
- Integración con Azure DevOps.
- Migración de la baseline de cualquier proyecto existente.
- Generación de un contexto técnico inicial a partir del código (cold start del contexto), según el default pendiente de Q16.
- Mantenimiento del contexto técnico en el carril ODD, salvo que el usuario acepte Q15.

### 2.3 Qué no cambia

Kickoff, mecánica de las compuertas (comandos y ledger), handoff, cierre de roles, `dependencies.archive`, skill `sdd-tasks`, claves obligatorias del status v2, almacén Engram y RDD. El estado de artefactos tampoco cambia si se confirma el default de Q10 (delta de contexto sin proyección en el status). Sí cambian `sdd-apply` y `sdd-verify` (C6, C7) y lo que evalúan las compuertas `design` y `role-apply`, que revisan además el delta de contexto.

## 3. Supuestos a confirmar

Supuestos de trabajo sobre los que se apoya esta propuesta. El usuario los confirma al revisarla; si alguno se rechaza, cambian el alcance de A, B o C y los criterios de éxito.

| ID | Supuesto | Consecuencia si se confirma |
|---|---|---|
| S1 | **Layout B'.** En `monorepo-decoupled` y `multirepo`, `workspaceRoot` pasa a ser la raíz del repo de specs y `openspec/` sigue literal debajo. `monorepo-embedded`, incluido este repo, resuelve exactamente a `<raíz>/openspec` sin cambio de comportamiento (R1). | Dos puntos de entrada en lugar de ~70 apariciones. `AllowedEditRoots` parte del repo de specs; los repos de código hermanos siguen el flujo existente de autorización por cambio (`axiom sdd-attempt grant`), sin pre-autorizar los repos de rol. |
| S2 | **Reparto de configuración.** La declaración estructural vive en `axiom.yaml` bajo `baseline:` y `technical_context:`, validada por el loader. Las reglas de conducta (`rules.<fase>`, `strict_tdd`, `artifact_store`) siguen en `openspec/config.yaml`. | `governance.context` no cambia. La frontera se documenta en `docs/openspec-config.md`. |
| S3 | **Nombre `baseline`**, no "profile", que ya significa otra cosa en `axiom init --profile`. | Claves `baseline:` y `technical_context:`. |
| S4 | **Contrato status v2 solo aditivo**: campos nuevos `omitempty` (R2). | `planningHome.path` sigue siendo la ruta absoluta del directorio `openspec` y `planningHome.mode` sigue en `repo-local`. |
| S5 | **Derivado por esta fase** de la exclusión de `axiom adopt`: la topología canónica (puntero `.axiom-workspace` y repo de specs) se monta a mano y se documenta; ningún comando la genera en inc-24. | A7 cubre la receta; `axiom init` no escribe punteros. |
| S6 | **Diseño del mantenimiento del contexto técnico**, propuesto para cumplir D5: dos modos en `technical_context:` (mantenido y de referencia); delta de contexto por cambio con "sin impacto" explícito; redacción en design, actualización en apply, aviso en verify e integración en archive con E2. | Reutiliza la maquinaria de integración de C (Q8) en lugar de añadir un mecanismo nuevo. Las rutas de referencia no se modifican nunca. |

## 4. Preguntas abiertas para spec y design

**Todas pendientes.** Cada una lleva un valor por defecto recomendado que no está decidido. Spec y design deben cerrarlas, o devolverlas al usuario a través del orquestador, antes de construir sobre ellas.

| ID | Pregunta | Default recomendado (pendiente) | Bloquea |
|---|---|---|---|
| Q3 | ¿Cómo se descubre la carpeta maestra desde un directorio anidado? | Walk-up en Go hasta el primer directorio con `.axiom-workspace`, `axiom.yaml` u `openspec/` (gana el más cercano; refina la inclinación del explorador de buscar solo el puntero, para no capturar un proyecto embebido anidado). Solo el puntero o un `axiom.yaml` con topología desacoplada o multirepo re-enraízan según B'. Desde el propio repo de specs, la carpeta maestra se deduce por el sufijo de `specs_repository` (o por `workspace.root`, hoy sin lector). El fallback de subdirectorios de `ResolveConfigFile` (`repo-specs`, `specs`, `openspec`, `.openspec`, `especificacion`) no mueve el planning home. Maestra con `openspec/` propio y puntero: rechazo. Los 17 comandos no cambian. Efecto a confirmar: `--cwd` en un subdirectorio de un proyecto embebido pasaría a resolver la raíz del proyecto en lugar de `<subdirectorio>/openspec`. | A |
| Q5 | Forma de la declaración y mapeo dominio → documento | Lista de documentos por dominio (`domain`, `path`, perfil de requisito); la documentación de solo lectura se declara como contexto de referencia (Q10), no en `baseline:`. Un dominio sin documento declarado se rechaza por defecto: nunca se crea `openspec/specs/<dominio>`. | B, C |
| Q6 | Formato del delta para una baseline del proyecto | Delta nativo (`## ADDED/MODIFIED/REMOVED Requirements`) con el ID en el encabezado del requisito; render al formato del proyecto por plantilla (p. ej. `### {id}: {title}`). | C |
| Q7 | IDs duplicados y asignación de IDs nuevos | Rechazar toda operación sobre un ID ambiguo y exigir normalización previa en el repo del proyecto. Spec usa IDs provisionales; archive asigna los definitivos para evitar colisiones con el flujo antiguo. Con identidad por ID, cambiar el título es un MODIFIED; renumerar queda fuera. | C |
| Q8 | Documentos no-requisito (modelo de datos, flujos, interfaces, integraciones, seguridad) | E2 (edición dirigida por agente con verificación nativa de presencia y alcance), coherente con D3, apoyado en el registro por ID de E1 como entrada de la verificación. Si el spike muestra que un documento carece de anclas estables, queda en E1 (pendiente de integrar) hasta que el proyecto lo normalice (E3). Traza en la carpeta del cambio. Modos descritos en `exploration.md` §4.3. El mismo mecanismo integra el delta de contexto en las rutas mantenidas (C4, D5): una sola maquinaria de integración en prosa. | C |
| Q9 | Coexistencia con el flujo antiguo en vuelo y registros del proyecto | Archive exige árbol de trabajo limpio en los documentos de baseline afectados, para que el diff de integración quede aislado y revertible; los conflictos con ediciones concurrentes se resuelven en git, no en Axiom. Archive no actualiza `REGISTRO_INCREMENTOS.md` ni `REGISTRO_BUGS.md` en inc-24. | C |
| Q10 | ¿Cómo se declaran los dos modos del contexto técnico, qué fases lo leen y con qué presupuesto, y qué forma tiene el delta de contexto? | Declaración: entradas por modo (mantenido y de referencia), con identificadores exactos en spec y design; una ruta no puede estar en ambos modos. Lectura: `sdd-explore`, `sdd-design`, `sdd-apply` y `sdd-verify` leen la entrada declarada de ambos modos y consultan el resto bajo demanda. Escritura: solo archive, solo en rutas mantenidas y solo a partir del delta; una ruta mantenida fuera del repo de specs exige la autorización por cambio existente (S1), nunca implícita. Delta: fichero propio del cambio (p. ej. `context-impact.md`) sin proyección en el status, para que apply no reabra un `design.md` ya aprobado y el estado de artefactos no cambie. Ausencia del delta con contexto mantenido declarado: se señala en la compuerta `design`, y archive se detiene y pide registrar el impacto o "sin impacto"; nunca lo asume. | B, C |
| Q11 | Guardas en `knowledge sweep`, `axiom archive coldstart` y el grabador cuando hay baseline del proyecto | Guardas mínimas en inc-24: `axiom init`, `sdd-init`, la rama sin baseline de `sdd-archive`, `axiom archive coldstart` y `sync`, `knowledge sweep` y el grabador rechazan, con continuación, crear `openspec/specs/` u `openspec/INDEX.md`. Las lecturas de catálogo (livingdoc, dashboard, knowledge) informan de que la baseline es externa; indexarla en esos catálogos queda fuera. | B |
| Q13 | Gate `integration` con evidencia `pr_merged` en multirepo | Documentar ahora que `pr_merged` se comprueba contra el grafo git del repo de specs (`internal/cli/sdd_gate_evidence.go:42-47`) y que la integración de los repos de código se acredita con `deployment` o `attestation`. Un flag `--repo` queda para un cambio posterior. | A |
| Q14 | **Detectada en esta fase.** Unificar la convención (A3) cambia livingdoc en proyectos embebidos con `specs_repository` distinto de `""`, `.` u `openspec`: hoy livingdoc lee `<raíz>/<specs_repository>/specs` y el motor, `<raíz>/openspec/specs`. | El motor manda (`<raíz>/openspec`) y livingdoc se alinea, con aviso cuando `<raíz>/<specs_repository>/specs` existe y difiere. | A |
| Q15 | **Carril ODD.** ODD no tiene archive y la mayoría de bugs e incrementos pequeños irán por ODD, así que el contexto mantenido derivaría ahí. ¿Se extiende el mantenimiento a ODD? | Regla mínima en el paso de cierre de ODD que genera `internal/components/agentguidance/routing.go` (paso 7, `:63`): si una unidad de trabajo cambia algo documentado en el contexto técnico mantenido, se actualiza en el mismo commit de esa unidad. Requiere decisión del usuario porque amplía el alcance fuera de SDD. | — (si se acepta, entra en C) |
| Q16 | ¿Qué ocurre si un documento mantenido declarado aún no existe? | Archive puede crearlo solo dentro de las rutas mantenidas declaradas y lo registra en la traza. Generar un contexto técnico inicial a partir del código queda fuera (cold start). | C |

Las preguntas Q1, Q2, Q4 y Q12 de la exploración quedan cubiertas por S1, S3 y D4; la decisión de entrega se toma en la fase de tareas (§9).

## 5. Capacidades (Capabilities)

> Contrato con `sdd-spec`. Cada capacidad nueva recibe una spec completa en `openspec/changes/inc-24-spec-baseline-profile/specs/<nombre>/spec.md`; cada capacidad modificada recibe una delta spec.

### 5.1 Nuevas capacidades (New Capabilities)

- `planning-home-resolution` (A): resolución de carpeta maestra, raíz del repo de specs y planning home desde cualquier directorio de partida; regla B' por topología; invariancia de `monorepo-embedded`; rechazo de ambigüedad; consumo idéntico por `axiom sdd status`, `continue`, `kickoff seal|show`, `gate record|show`, `sdd-attempt grant`, handoff, `role barrier` y los consumidores de catálogo (livingdoc, knowledge, dashboard, TUI); semántica de `workspaceRoot` y `planningHome` en el status; regla de ejecutores para rutas `openspec/...`.
- `spec-baseline-declaration` (B): bloque `baseline:` (documentos por dominio y perfil de requisito; la documentación de solo lectura se declara como contexto de referencia en `project-technical-context`), defecto OpenSpec, validación, proyección aditiva en el status v2 y guarda transversal: ningún creador produce una segunda baseline cuando la declarada es del proyecto. Incluye `sdd-init` y knowledge, que no tienen spec viva propia (§12).
- `project-technical-context` (B, C): declaración, lectura y mantenimiento del contexto técnico. En B: bloque `technical_context:` con dos modos (mantenido y de referencia), validación, proyección aditiva en el status v2 y fases lectoras. En C: delta de contexto por cambio con "sin impacto" explícito (redactado en design, actualizado en apply), aviso no bloqueante de verify y garantía de que las rutas de referencia nunca se modifican.
- `spec-baseline-integration` (C): compose sobre la baseline declarada (perfil de encabezado, identidad por ID, plantilla de render, duplicados, frontera por nivel, fidelidad de bytes), resolución por dominio en `axiom sdd archive-compose`, integración de documentos no-requisito con su traza y lectura de la baseline por `sdd-explore`, `sdd-propose` y `sdd-spec`. El mecanismo E2 (edición dirigida por agente, verificación nativa por diff y traza por entrada) se especifica aquí una sola vez; la integración del contexto lo reutiliza.

### 5.2 Capacidades modificadas (Modified Capabilities)

- `workspace-topology`: REQ-2.1 (esquema con bloques opcionales `baseline` y `technical_context`); REQ-1.2 (`axiom workspace validate` los valida); REQ-2.3 y REQ-2.4 (patrón canónico con `.axiom-workspace` y planning home en `<repo de specs>/openspec`).
- `living-documentation`: REQ-1.1 y REQ-1.2 (raíz e `INDEX.md` bajo el planning home resuelto; ver Q14); REQ-2.1, REQ-3.1 y REQ-3.4 (`axiom archive coldstart` y `sync` no crean una segunda baseline cuando la declarada es del proyecto).
- `sdd-archive-lifecycle`: REQ-21.17 (archive integra en la baseline declarada: la del proyecto si existe, `openspec/specs/` por defecto; nunca crea una segunda. Además integra el delta de contexto en las rutas mantenidas con el mecanismo E2 de `spec-baseline-integration`, registra la traza y nunca modifica las rutas de referencia); REQ-21.16 (el escenario de rechazo pasa a referirse a la baseline declarada y, solo si Q13 cambia, a la evidencia `pr_merged` en multirepo).
- `sdd-block-review-gates`: REQ-21.9 (la compuerta `design` evalúa además el impacto previsto en el contexto mantenido) y REQ-21.11 (la compuerta `role-apply` evalúa además el impacto final).
- `multi-project-hub`: REQ-2.2 (`axiom init` no crea `openspec/specs/` cuando `axiom.yaml` ya declara una baseline del proyecto).

Ubicación de la integración del contexto (decisión de esta fase): va en el delta de `sdd-archive-lifecycle` y no en `spec-baseline-integration`, porque REQ-21.17 es el requisito que enumera todo lo que archive actualiza. Así el contrato de archive queda en un solo sitio, `spec-baseline-integration` no se amplía más allá de la spec (el contexto no es baseline) y el mecanismo E2 se especifica una sola vez y se reutiliza.

Sin delta, comprobado en esta fase: `dashboard-sdd-orchestration`, `local-web-dashboard`, `structured-handoffs`, `multi-role-fan-out`, `sdd-lane-kickoff` y `axiom-sdd-cli-integration` citan rutas relativas `openspec/changes/...` que, bajo B', siguen siendo válidas respecto a la raíz resuelta. Condicionadas a Q11: `tui-ui-parity` (`ScreenLivingDoc`) y `living-documentation` REQ-4.1 y REQ-4.2, solo si las lecturas de catálogo deben avisar de una baseline externa.

## 6. Enfoque (Approach)

Orden A → B → C. Cada rebanada deja el sistema verde y entregable por sí misma.

### 6.1 A: re-enraizar, no hilar

- **Resolver nuevo en `internal/workspace`**, paquete solo-fork (sin equivalente upstream y fuera del ratchet de refusals), apoyado en `ResolveConfigFile` y `LoadConfig`, que ya resuelven el puntero.
- **Dos puntos de entrada delegan en él.** El resto de hojas sigue construyendo `<workspaceRoot>/openspec/...`, ahora con `workspaceRoot` = raíz del repo de specs. Se evita hilar un `PlanningHome` por ~70 apariciones y se conservan sin generalizar los grants, el ledger en el git common dir, `RefuseArchivedRoot` y `PrepareChangeInstanceConsent` (R3).
- **Una sola convención** para livingdoc, knowledge, dashboard y TUI.
- **Ejecutores:** regla explícita en `sdd-phase-common.md` y bloques shell con `cd` a la raíz resuelta. Es una hipótesis: se valida con un spike en Claude Code y OpenCode, porque el cwd de bash se reinicia entre llamadas (R17).
- **Red de seguridad:** test de equivalencia del layout embebido (`specs_repository` en `""`, `.`, `openspec` y `specs`) y tests actuales de rutas sin tocar sus aserciones.

### 6.2 B: declarar en `axiom.yaml`, proyectar en el status

- `WorkspaceConfig` gana `Baseline` y `TechnicalContext` opcionales. El YAML no es estricto, así que los binarios anteriores ignoran los bloques nuevos.
- El status v2 proyecta los localizadores como campos `omitempty`, con el mismo patrón que `Consent`, `Archived`, `Governance`, `GateQuestion` y `PhaseInstructions`. Las skills los leen del status ("never re-resolve") en lugar de parsear YAML.
- Todas las guardas usan la misma consulta al resolver ("¿la baseline declarada es del proyecto?"). Los refusals nuevos viven en `internal/workspace` o nombran una continuación, para que la línea base del ratchet no crezca.
- `TechnicalContext` declara los dos modos (mantenido y de referencia). En B las fases solo leen el contexto; el mantenimiento llega completo con C.

### 6.3 C: generalizar el compose, no sustituirlo

- `ComposeOptions` gana un perfil de encabezado opcional; con `nil` se conserva el comportamiento nativo byte a byte (los 10 tests de compose y los 3 de CLI son la red).
- Identidad por ID, rechazo de duplicados ambiguos, frontera por nivel, plantilla de render y validación completa antes de escribir: o se aplican todas las operaciones o ninguna.
- `archive-compose` resuelve documento y perfil desde `axiom.yaml`; `sdd-archive` deja de plantillar rutas en shell (sus bloques están fijados y se ejecutan en `internal/assets/assets_test.go`).
- El detalle de las skills va a `references/`, para contener la divergencia con upstream y el presupuesto de cuerpo de `docs/skill-style-guide.md`.
- **Mantenimiento del contexto con la misma maquinaria.** El delta de contexto reutiliza E2 (Q8): design redacta el impacto, apply lo actualiza, verify avisa de forma heurística y archive lo integra con verificación nativa por diff y traza por entrada. Solo cambian el destino (rutas mantenidas) y las rutas vetadas (las de referencia).
- **Por qué el delta va en C y no en B.** Un cambio archivado es inmutable (REQ-21.18): si design y apply redactaran deltas antes de que archive supiera integrarlos, esos cambios se archivarían sin integrar y no podrían recuperarse. En C, el ciclo del delta se entrega completo.

## 7. Prerrequisito de C: spike del proyecto de referencia

Análisis de solo lectura de la baseline del proyecto de referencia antes del diseño detallado de C (`exploration.md` §8.4): encabezados por nivel y documento; cobertura de `^### RF-\d+:`; IDs repetidos, distinguiendo repetición idéntica de conflicto; `###` no-requisito dentro de bloques de requisito; CRLF o LF y BOM; longitud de bloques; anclas estables en los documentos 03–07 y en `context/`, que anclarán el delta de contexto.

- Los datos los aporta el usuario o se obtienen con acceso explícito autorizado a esa ruta. Ninguna fase lee rutas fuera de este repositorio sin esa autorización.
- Sin spike, C solo puede diseñarse contra fixtures sintéticos, y Q7 y Q8 quedan sin evidencia. Las cifras del proyecto de referencia citadas en esta propuesta (512 encabezados `### RF-NNN`, 496 IDs distintos, 16 repeticiones) son datos del usuario, no contrastados.
- Dato aportado por el coordinador, no contrastable desde este repositorio: el flujo actual del proyecto de referencia ya escribe documentación técnica por rol tras la revisión de código, así que mantener el contexto coincide con la práctica del proyecto adoptado.

## 8. Áreas afectadas (Affected Areas)

Paquetes `internal/` afectados: `workspace`, `sddstatus`, `cli`, `livingdoc`, `knowledge`, `hub`, `dashboard`, `tui`, `multirole` y `assets`; por evaluar: `kickoff` y `reviewtransaction`; por decidir (Q15): `components/agentguidance`. Fuera de `internal/`: `cmd/axiom`, `testdata/golden` y `docs`.

| Área | Impacto | Reb. | Qué cambia |
|---|---|---|---|
| `internal/workspace/` (`loader.go`, `types.go`, `validator.go`) | Nuevo y modificado | A, B | Resolver de raíz; tipos y validación de `baseline` y `technical_context` (dos modos) |
| `internal/sddstatus/status.go` | Modificado | A | `resolveWorkspaceRoot` delega en el resolver; literales de mensaje |
| `internal/sddstatus/status_v2.go` | Modificado | B | Campos aditivos `omitempty` |
| `internal/sddstatus/openspec_archive_compose.go` | Modificado | C | Perfil de encabezado, identidad por ID, render, duplicados, frontera por nivel |
| Verificador nativo E2 (ubicación en design) | Nuevo | C | Presencia y alcance por diff y traza por entrada; compartido por los documentos no-requisito de la baseline y por el contexto mantenido; veta las rutas de referencia |
| `internal/cli/sdd_governance.go`, `sdd_kickoff.go`, `sdd_gate_evidence.go` | Modificado | A | Raíz desde el resolver; carga de configuración sin `os.Stat` previo; documentación de `pr_merged` (Q13) |
| `internal/cli/sdd_archive_compose.go` | Modificado | C | `--cwd`, `--document <dominio>`; mensajes |
| `internal/kickoff/archived.go` | Sin cambio previsto | A | Verificar que el prefijo `openspec/changes/archive` sigue siendo válido bajo B' |
| `cmd/axiom/main.go` | Modificado | A, B | Resolvers de cambio, handoff y `role barrier`; cargas de configuración; comandos `archive` y `knowledge` |
| `internal/livingdoc/` (`service.go`, `synthesizer.go`, `indexer.go`) | Modificado | A, B | Raíz e `INDEX.md` desde el resolver; guarda |
| `internal/knowledge/` (`planner.go`, `query.go`, `sweep.go`, `index.go`, `finalize.go`, `recorder.go`) | Modificado | A, B | `SpecsRoot` desde el resolver; guarda |
| `internal/hub/` (`init.go`, `detector.go`) | Modificado | B | Sin andamiaje de `openspec/specs/` con baseline del proyecto |
| `internal/dashboard/service.go`, `internal/tui/` | Modificado | A | Rutas de cambios, archivo y acumulativo desde el resolver |
| `internal/multirole/barrier.go` | Modificado | A | Textos de ruta |
| `internal/reviewtransaction/risk.go` | Por evaluar | A | Clasificación por segmento `openspec` (R14); solo afecta a RDD |
| `internal/assets/skills/sdd-archive/SKILL.md` | Modificado | C | Integración en la baseline declarada; rama sin baseline; integración del delta de contexto |
| `internal/assets/skills/sdd-{spec,propose,explore,onboard}/SKILL.md` | Modificado | C | Lectura de la baseline por localizador |
| `internal/assets/skills/sdd-{design,apply}/SKILL.md` | Modificado | C | Redacción y actualización del delta de contexto |
| `internal/assets/skills/sdd-verify/SKILL.md` | Modificado | C | Aviso no bloqueante cuando el diff contradice un "sin impacto" |
| `internal/assets/skills/sdd-init/` | Modificado | B | No crear `openspec/specs/` con baseline del proyecto |
| `internal/assets/skills/_shared/` (`sdd-phase-common.md`, `openspec-convention.md`, `sdd-status-contract.md`, `persistence-contract.md`) | Modificado | A, B, C | Regla de ejecutores, convención y campos del status |
| `internal/assets/claude/sdd-orchestrator-workflow.md` y el resto de plantillas de orquestador | Modificado | C | Las compuertas `design` y `role-apply` presentan el delta de contexto |
| `internal/components/agentguidance/routing.go` y sus tests | Por decidir (Q15) | C | Regla ODD: actualizar el contexto mantenido en el mismo commit de la unidad de trabajo |
| Definiciones de agentes `sdd-*` en assets, `internal/assets/assets_test.go` y `testdata/golden/` | Modificado | B, C | Terminología "baseline"; pins; regeneración de goldens |
| `docs/manual-de-inicio.md`, `docs/openspec-config.md` | Modificado | A, B | Receta multirepo canónica; frontera entre `axiom.yaml` y `openspec/config.yaml` |

## 9. Tamaño y entrega

| Rebanada | Estimación sin medir (líneas cambiadas) | Unidades de 400 líneas |
|---|---|---|
| A | 800–1.200 | 2–3 |
| B | 850–1.350 | 3–4 |
| C | 1.600–2.700 | 4–7 |
| **Total** | **3.250–5.250** | **9–14** |

- Fuente: `exploration.md` §8.1 para A. B y C se reestiman en esta revisión, también sin medir: B suma los dos modos del contexto (+50–150 líneas) y C suma el delta de contexto en design y apply, el aviso de verify, la integración en archive sobre el mecanismo E2, sus pins y sus goldens (+400–800). Si se acepta Q15, hay que añadir 30–80 líneas. Las cifras incluyen código, tests, assets y goldens. Los goldens regenerados no cuentan para el presupuesto de autoría, así que el recuento efectivo será algo menor, pero cada rebanada sigue muy por encima de 400 líneas.
- La estrategia de entrega de la sesión es `single-pr`. Con esta previsión, `sdd-apply` no puede empezar sin una de dos resoluciones: PRs encadenados por rebanada (A → B → C; 2–3 PRs para A, 3–4 para B y 4–7 para C) o una `size:exception` explícita para un único PR.
- **Esta propuesta no decide la entrega.** La decide el orquestador en la fase de tareas con la guarda de carga de revisión; `sdd-tasks` debe emitir las líneas `Decision needed before apply`, `Chained PRs recommended` y `400-line budget risk`.
- Palanca de tamaño, no decidida: dashboard y TUI son la parte de A que la exploración marcó como diferible.

## 10. Riesgos (Risks)

IDs R1–R14 heredados de la exploración; R17–R21, nuevos de esta fase; R22–R27, de la revisión por D5.

| ID | Riesgo | Prob. | Mitigación |
|---|---|---|---|
| R1 | Regresión del layout embebido (este repo y todo proyecto embebido) | Media | Resolución exacta a `<raíz>/openspec`; test de equivalencia; tests de rutas sin tocar sus aserciones; `bench/` ejecutado aparte |
| R2 | Contrato del status "congelado externo", compartido con consumidores externos | Media | Solo campos `omitempty`; `status_v2_clean_break_test` como guardia |
| R3 | Autoridad de edición y ledger ligados a git | Baja con S1 | El repo de specs es git; los repos de código, por autorización por cambio |
| R4 | IDs duplicados en la baseline del proyecto | Alta | Rechazo de operaciones ambiguas y normalización previa (Q7) |
| R5, R6 | Frontera de bloque y fidelidad CRLF/BOM al cambiar el patrón de encabezado | Media | Frontera por nivel; fixtures LF, CRLF y BOM antes de implementar |
| R7 | Segunda baseline accidental | Media | Guarda transversal con una sola consulta (Q11); test de no existencia de `openspec/specs/` |
| R8 | Ediciones concurrentes con el flujo antiguo e IDs nuevos que colisionan | Media | IDs definitivos en archive; árbol limpio como precondición (Q7, Q9) |
| R9 | Más divergencia con upstream en `sddstatus`, `cli` y skills | Media | Lógica en `internal/workspace`; costuras mínimas; detalle en `references/` |
| R10 | Ratchets de CI: refusals, deadcode, pins y goldens | Alta | Refusals con continuación; todo cableado desde `cmd/axiom`; pins y goldens en la misma rebanada |
| R11 | `pr_merged` en multirepo comprueba el repo de specs, no el de código | Media | Documentar (Q13) |
| R12 | Proyecto Engram inferido desde una raíz no git | Baja con S1 | El repo de specs es git; comprobar en design |
| R13 | Entrega muy por encima de 400 líneas con `single-pr` | Alta | Decisión del orquestador en la fase de tareas (§9) |
| R14 | RDD clasifica como operativo un `.md` bajo un segmento `openspec`; las ediciones de la baseline del proyecto quedarían como pasivas | Baja | Solo con RDD activado (opt-in); evaluar en design |
| R17 | Los ejecutores resuelven `openspec/...` contra el cwd y no contra el `workspaceRoot` | Media | Regla explícita y spike en Claude Code y OpenCode antes de cerrar A |
| R18 | El walk-up captura el proyecto equivocado (p. ej. un proyecto embebido dentro de una maestra) | Media | Gana el más cercano; B' solo re-enraíza con puntero o topología desacoplada o multirepo (Q3) |
| R19 | inc-23 sin archivar: sus capacidades `knowledge-*` no existen en `openspec/specs/` | Media | Guardas de knowledge dentro de `spec-baseline-declaration`, o archivar inc-23 antes de la fase de spec (§12) |
| R20 | Con E2, archive pasa de copia mecánica a edición asistida, también en el contexto mantenido; la asignación de modelo actual de `sdd-archive` puede quedarse corta | Media | Decidir en design el actor de la edición y la verificación nativa (Q8, C4) |
| R21 | El spike del proyecto de referencia no llega a tiempo | Media | C se diseña contra fixtures sintéticos; no se aplica C sin el spike |
| R22 | El delta de contexto se omite o es erróneo (p. ej. "sin impacto" cuando sí lo hay) y el contexto mantenido se desactualiza | Media | "Sin impacto" explícito obligatorio; revisión en las compuertas `design` y `role-apply`; aviso heurístico de verify; archive no asume la ausencia (Q10) |
| R23 | Errores al editar la prosa del contexto mantenido, o escrituras fuera de alcance | Media | Verificación nativa E2 de presencia y alcance por diff; traza por entrada; revertible con git |
| R24 | Escritura accidental en rutas de referencia | Baja | Fuera del conjunto permitido por la verificación por diff; test dedicado |
| R25 | Deriva del contexto por cambios en el carril ODD, que no tiene archive | Alta si Q15 no se acepta | Regla mínima en el cierre de ODD (Q15, pendiente) |
| R26 | Falsos positivos o negativos del aviso heurístico de verify | Media | Solo informativo; heurística acotada a manifiestos de dependencias, módulos de primer nivel y comandos de build o CI; nunca bloquea archive |
| R27 | Rutas mantenidas fuera del repo de specs (p. ej. documentación por rol en repos de código) | Media | Autorización por cambio existente (S1), nunca implícita; el diff verificado cubre cada repo afectado (Q10) |

Riesgo global de la propuesta: **alto**, por R1, R4, R10 y R13.

## 11. Plan de reversión (Rollback Plan)

- **Por configuración.** Todo lo nuevo es opt-in: sin `.axiom-workspace` y sin bloques `baseline:` ni `technical_context:`, el resultado es idéntico al actual (primer criterio de §13). Un proyecto vuelve atrás retirando los bloques de `axiom.yaml` (vuelve a OpenSpec por defecto) o retirando el puntero y ejecutando los verbos desde la raíz del repo de specs, donde hoy ya se resuelve `<cwd>/openspec`. Para desactivar solo el mantenimiento del contexto basta con declarar sus rutas como de referencia: las fases siguen leyéndolo y archive deja de escribir en él.
- **Por código.** Cada rebanada se entrega en commits o PRs revertibles con `git revert`, en orden inverso C → B → A, porque B y C dependen de A. Los campos del status son `omitempty`: retirarlos no rompe a los consumidores que toleran su ausencia.
- **Por datos.** No hay migraciones. Una integración en la baseline del proyecto es un diff git en el repo de specs, revertible con `git revert`. C2 exige que el compose valide todas las operaciones antes de escribir, de modo que un rechazo no deja la baseline a medias, y la precondición de árbol limpio propuesta en Q9 aísla el diff. Lo mismo vale para el contexto mantenido: cada integración es un diff git revertible con traza por entrada, y las rutas de referencia nunca reciben escrituras, así que en ellas no hay nada que revertir. La carpeta archivada del cambio es inmutable (REQ-21.18): una integración errónea se corrige con un bug o un incremento nuevo.
- **Señal para revertir una rebanada:** fallo, tras la fusión, del test de equivalencia embebido, de `status_v2_clean_break_test` o de los tests nativos de compose, o cualquier escritura detectada en rutas de referencia.

## 12. Dependencias (Dependencies)

- **Orden interno A → B → C.** B usa el resolver; C usa la declaración y su proyección.
- **Spike de ejecutores** (cwd de sub-agentes) en Claude Code y OpenCode, antes de cerrar A.
- **Spike del proyecto de referencia** (§7), antes del diseño detallado de C. Requiere datos del usuario o acceso autorizado.
- **inc-23-knowledge-profile**: implementado y verificado (`verify-report.md` en PASS), pero sin archivar. Toca `internal/knowledge` e `internal/hub`, y sus capacidades `knowledge-profile-init`, `knowledge-sweep-command` y `knowledge-query-engine` aún no están en `openspec/specs/`. Recomendación de esta fase: especificar las guardas de knowledge dentro de `spec-baseline-declaration`; la alternativa es archivar inc-23 antes de la fase de spec para poder escribir deltas contra esas capacidades. Decide el orquestador.
- **Carril ODD (Q15):** si el usuario acepta la regla, el cambio toca `internal/components/agentguidance/routing.go` y sus tests y goldens, fuera del carril SDD.
- Sin dependencias externas nuevas previstas.

## 13. Criterios de éxito (Success Criteria)

- [ ] **Embebido intacto.** En este repo (`monorepo-embedded`, `specs_repository: "openspec"`, sin puntero), `axiom sdd status --json` lanzado desde la raíz devuelve el mismo `workspaceRoot` que hoy y `planningHome.path` = `<repo>/openspec`. Un test de equivalencia cubre `specs_repository` en `""`, `.`, `openspec` y `specs`, y los tests existentes de `sddstatus`, `cli`, `kickoff` y `livingdoc` pasan sin modificar sus aserciones de ruta.
- [ ] **Multirepo canónico.** En un fixture con `<maestra>/.axiom-workspace` → `repo-specs/axiom.yaml` (`multirepo`) y repos de código hermanos, partiendo de `<maestra>`, de `<maestra>/repo-specs` y de un subdirectorio de un repo de código, `axiom sdd status`, `continue`, `kickoff seal|show`, `gate record|show`, `sdd-attempt grant`, handoff y `role barrier` resuelven el mismo planning home `<maestra>/repo-specs/openspec`, con `workspaceRoot` = `<maestra>/repo-specs`.
- [ ] **Ambigüedad rechazada.** Una maestra con `openspec/` propio y puntero produce un rechazo que nombra la continuación y no escribe nada.
- [ ] **Contrato del status.** `status_v2_clean_break_test` pasa; los campos nuevos solo aparecen cuando existe la declaración correspondiente.
- [ ] **Declaración validada.** `axiom workspace validate` acepta una declaración válida y rechaza, con mensaje en español y código de salida `1`, un patrón sin grupos `id` y `title`, una identidad desconocida o un documento inexistente.
- [ ] **Sin segunda baseline.** Con una baseline del proyecto declarada, tras `axiom init`, la rama sin baseline de `sdd-archive`, `axiom archive coldstart` y `sync`, `knowledge sweep` y el grabador, no existen `openspec/specs/` ni `openspec/INDEX.md` (aserción por test); la skill `sdd-init` lo prescribe (pin de texto). Archive rechaza crear una segunda baseline.
- [ ] **Round-trip byte a byte.** Sobre una baseline fixture con encabezados `### RF-NNN: <título>`, en variantes LF, CRLF y con BOM, y con subsecciones `###` no-requisito, un delta con ADDED, MODIFIED y REMOVED produce una salida idéntica byte a byte a la original salvo en los bloques editados.
- [ ] **Duplicados.** Una operación sobre un ID repetido se rechaza nombrando el ID y sus ocurrencias, sin escribir el documento.
- [ ] **Compose nativo intacto.** Los 10 tests de `openspec_archive_compose_test.go` y los 3 de `internal/cli/sdd_archive_compose_test.go` pasan sin cambios.
- [ ] **Skills de lectura.** `sdd-explore`, `sdd-propose` y `sdd-spec` toman la baseline del localizador del status cuando hay declaración (pin de texto), sin rutas `openspec/specs/` fijas.
- [ ] **Documentos no-requisito** (según Q8). Cada ID del delta figura en la traza del cambio; con E2, la verificación nativa rechaza un diff que toque documentos no declarados.
- [ ] **Contexto técnico: modos y lectura.** `axiom workspace validate` valida los dos modos de `technical_context:` y las fases lectoras de Q10 reciben ambos localizadores desde el status.
- [ ] **Mantenimiento del contexto.** Con un delta de contexto que modifica una sección de un documento mantenido, archive actualiza ese documento según el delta y registra la traza por entrada en la carpeta del cambio y en el informe de archive.
- [ ] **Referencia intacta.** Ninguna fase modifica las rutas de referencia: tras design, apply, verify y archive, el diff no contiene cambios en ellas (comprobación por diff, con test).
- [ ] **"Sin impacto" explícito.** Un delta "sin impacto" deja el contexto mantenido intacto (diff vacío en sus rutas) y queda registrado como decisión; un cambio sin delta de contexto nunca pasa en silencio.
- [ ] **Aviso de verify.** Con un delta "sin impacto" y un diff que toca un manifiesto de dependencias, `sdd-verify` emite el aviso no bloqueante y archive no queda bloqueado por él.
- [ ] **Compuertas.** La presentación de las compuertas `design` y `role-apply` incluye el delta de contexto (pin de texto del orquestador).
- [ ] **Documento mantenido inexistente** (según Q16). Archive lo crea solo dentro de las rutas mantenidas declaradas y lo registra en la traza.
- [ ] **Carril ODD** (solo si se acepta Q15). La guía ODD generada por `routing.go` incluye la regla de actualizar el contexto mantenido en el mismo commit de la unidad de trabajo (test o golden).
- [ ] **CI verde.** `go test ./...` y `go vet ./...` pasan; la línea base del ratchet de refusals no crece; el ratchet de deadcode pasa; los goldens están regenerados.
- [ ] **Spike del proyecto de referencia** documentado antes de aplicar C.
