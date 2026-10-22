# Exploración: inc-25-ramas-y-worktrees-por-topologia

> Cambio `inc-25-ramas-y-worktrees-por-topologia` · fase sdd-explore · base origin/main 17d4029e (rama inc/inc-25-ramas-y-worktrees-por-topologia). Entradas cerradas: D1-D10 (Engram discovery/worktree-policy-por-topologia-decisiones #619; auditoría #617). Rutas relativas a la raíz del worktree. Números de línea verificados contra el código actual. Copia en Engram: `sdd/inc-25-ramas-y-worktrees-por-topologia/explore` (#634).

## 1. Estado actual (verificado)

### 1.1 Topologías (D1)
- internal/workspace/types.go:6-15 constantes; monorepo-decoupled en :10-11. WorkspaceSection :18-28, RoleConfig :36-41 (Repositories []RepositoryEntry), WorkspaceConfig :56-62: no hay claves git: ni repositories: de primer nivel.
- loader.go:120-151 validateBaseConfig acepta las tres (:125-131). parseLayered :59-96 ya decodifica desde yaml.Node (:64,:75) y deja fuera claves desconocidas: añadir git:/repositories: es aditivo para binarios antiguos.
- validator.go:66-105: casos decoupled (:78-90) y multirepo (:92-104) casi idénticos. :107-124 valida roles.<k>.repositories[].path solo con IsDir (no exige git ni raíz git).
- locate.go:398-407 separatesSpecs trata decoupled y multirepo igual; Roots :414-423; deduceMaster :454-485. En comportamiento no hay diferencia entre decoupled y multirepo: D1 elimina un valor de enumeración y mensajes.
- Superficies de usuario que ya solo ofrecen embedded|multirepo: cmd/axiom/main.go:1801,1942; dashboard/assets/index.html:663-664; hub/types.go:73. Residuos: 6 sitios en internal/workspace/* y sddstatus/planning_home_test.go:183.
- Specs/docs con decoupled: openspec/specs/workspace-topology (:82,97,135,145,150), planning-home-resolution (:18,31,129,139,183,255), spec-baseline-declaration:94, sdd-archive-lifecycle:17, local-web-dashboard:15; docs/manual-de-inicio.md:120,158,189,248; axiom.example.yaml:13,19.
- Efecto D1 sin alias: axiom.yaml con monorepo-decoupled falla en validateBaseConfig ("topología desconocida").

### 1.2 Hub (D7)
- hub/types.go:10-19 WorkspaceRecord (sin campos de worktree); InitOptions :70-77. manager.go:117-207 Register(path,name,topology) posicional; rechaza worktrees (:123-125). Llamadores: hub/init.go:284, dashboard/service.go:70 y :836, cmd/axiom/main.go:1957 + pruebas: preferible método aparte (SetWorktreePrefs) o campos omitempty.
- hub/detector.go:406-446 IsGitWorktree: cualquier segmento axiom-wt (:416-422) o .git fichero. Heurística con nombre propio; la raíz de worktrees por defecto del producto no debe volver no registrable el proyecto.
- hub/init.go:128-309 Init: topología por defecto embebida (:151-153); perfiles knowledge/spec-only fuerzan multirepo con rol único cuyo repo es "." (:201-211): el repo del rol es el propio repo de specs.
- Dashboard: server.go:224-242 POST /api/projects/init; service.go:819-861 AddProject/InitProject; app.js:914-952. GET /api/workspace solo lectura (server.go:281-293); no existe endpoint de escritura sobre axiom.yaml.

### 1.3 Kickoff y evidencia
- kickoff/seal.go:60-111 Seal una sola vez (PublishFileNoReplace); types.go:119-126; schema.go:65-74 ParseKickoff con yaml.Unmarshal sin KnownFields: bloque git: adicional es compatible hacia atrás.
- cli/sdd_kickoff.go:42-77 runSDDKickoffSeal -> resolveGovernanceChangeRoot (cli/sdd_governance.go:34-54: workspace.Locate(cwd), la carpeta del cambio debe existir ya). No consulta git.
- kickoff/args.go:303-328,464-466: BaseRef por defecto "main" en pr_merged. Ascendencia local: kickoff/integration.go:44-58 -> cli/sdd_gate_evidence.go:19-45 -> reviewtransaction/snapshot.go:949-976 RevisionIsAncestor (merge-base --is-ancestor; "nunca consulta remoto").
- Otro literal main: dashboard/service.go:1643. Rama por defecto remota solo en reviewtransaction/gate.go:325-337 (ls-remote --symref).

### 1.4 Estado SDD y autoridad de edición
- sddstatus/status.go:145-149 ActionContext; :165-223 Status; resolveByPreferenceOrder :558-724 lista cambios activos del árbol actual (:338-349,:564-565), concesiones :644-655, applyEditAuthorityBlock con [workspaceRoot]+grantedRoots (:662), baseStatus :1366.
- sddstatus/edit_authority.go:88-127 detectUnauthorizedEditRoots resuelve rutas de tasks.md contra workspaceRoot, las reduce a raíz git (gitRootOf :263-278). Razón tipada: literal blocked(edit_authority_missing) (:292-300), único prefijo blocked(...) tipado hoy. Consentimiento: edit_authority_consent.go:233-278.
- No hay concepto de rama en sddstatus: blocked(wrong_branch) sería el segundo motivo tipado.

### 1.5 Escritores Go de la carpeta del cambio (D9-A) (~8 sitios)
dashboard/service.go:963-1054 CreateIncrement (MkdirAll :991, proposal.md :1043-1045; también lo usa axiom change create, cmd/axiom/main.go:504-554); service.go:864-953 migración al acumulativo (:922); service.go:1123 CreateHandoff; handoff/writer.go; kickoff/seal.go:60; kickoff/ledger.go:101; sddstatus/edit_authority_consent.go (marcador de instancia); cli/sdd_integrate.go (integration.yaml); cli/sdd_archive_compose.go:316; tui/model.go:6080-6110.
No hay escritor Go de odd/tasks/<feature>.md (internal/odd retirado en INC-20; odd-living-document/spec.md:11-35): ODD es solo doctrina de agente.

### 1.6 Dashboard y TUI
- dashboard/service.go:97-116 planningRoot() = Locate(ruta registrada); GetIncrements :188-206, FindIncrementPath :430-456, requireActiveIncrement :464-474 leen solo el checkout principal; tui/model.go:5988-6003 igual. Con D5 los cambios en curso viven en worktrees y estos listados los verían vacíos.
- Sumideros no archivados por diseño: e2e-cumulative (filtrado por nombre service.go:199, model.go:5991) y changes/cumulative-<rol> (service.go:922).
- Precedentes remotos en dashboard: service.go:1610-1683 GetSpecsSyncStatus ejecuta git fetch --quiet con tope 4 s fail-open (:1660-1667); :1686-1715 git pull --ff-only (acción explícita).

### 1.7 Doctrina y skills
- agentguidance/routing.go:48-64 protocolo ODD; :54 paso 0; :61 paso 5 (odd/tasks y espejo); :62 "branch first when on the default branch"; :113. remote_authorization.go:10-16 inyecta el contrato de autorización remota.
- sdd-apply/SKILL.md:118-132 Paso 2c (cuestionario worktree, git worktree add -b feat/{change}-{rol} ../wt-{change}-{rol} origin/<base>), :270-280 ODD-5.6 (PR, scripts/cleanup-worktree.sh), :301-303 reglas ODD-5.1/5.5/5.6. sdd-spec/SKILL.md:248,269 y sdd-design/SKILL.md:192,208 repiten ODD-5.1 (specs siempre en main). sdd-explore y sdd-archive NO mencionan ramas/worktrees (archive solo allowedEditRoots :87 y copia por shell :93).
- Ni assets del producto ni routing.go contienen hoy EnterWorktree/ExitWorktree: la prohibición solo vive en CLAUDE.md:23, AGENTS.md:24, GEMINI.md. D10 exige añadirla al producto (punto único: routing.go).
- assets/<runtime>/sdd-orchestrator.md (12 ficheros) solo mencionan worktrees en una frase de delegación.

### 1.8 Hooks por runtime (D10)
components/sdd/inject.go: Claude (:1866-1906) escribe en settings GLOBAL (adapter.SettingsPath(homeDir)): UserPromptSubmit (:2057), PreToolUse matcher Agent fail-closed (:2214-2274), Stop/SessionStart (:2287-2353). Codex (:1908-1975) hooks.json solo SessionStart. Kiro: .kiro/hooks (agents/kiro/adapter.go:133-137). OpenCode: plugins TS propios; carril host real usa tool.execute.before/after (scripts/crosslane/hostopencode.go:13-26) y cita "hook throws" tragados. Pi: sin hooks de Axiom (hostpi.go:245-253 extensión en print mode sin demostrar). capabilitymanifest no modela hooks.

### 1.9 RDD y worktrees (apartado g)
cli/review_atomic_start_integration_test.go:109-146,167-228: worktrees enlazados en rutas temporales arbitrarias, linajes independientes por worktree; review_start_lineage.go:41-60 deriva linaje de identidad de worktree; review_facade.go:1047-1058 rechaza usar un linaje desde otro worktree; autoridad en <GitCommonDir>/gentle-ai/... (review_mode.go:656; rdd_mode.go:805) compartida por el clon. snapshot.go:763,860-880 solo excluye worktrees anidados. Sin tratamiento de .claude/worktrees.

### 1.10 Dogfooding (apartado h)
- Cambios activos en main: openspec/changes/e2e-cumulative (sumidero vacío) e inc-23-knowledge-profile (kickoff continuo sellado, verify PASS, sin archivar).
- scripts/axiom-worktree.ps1/.sh: new|pr|done por slug; raíz <padre del principal>/axiom-wt/<slug> (.ps1:37-53); fetch por defecto (:93-101); prefijos feat|inc|fix|refactor (:25); pr empuja y abre PR --base main (:114-161); done desmonta, ff-only de main y axiom project remove (:163-221).
- scripts/claude-hooks/guard-primary-checkout.ps1: política local por RUTA (no por rama): deniega Write|Edit|NotebookEdit en el checkout principal (:125-143) y avisa tras Bash|PowerShell si el principal sale de main (:145-176); registro en .claude/settings.json local no versionado. CLAUDE.md:3 dice "en otros proyectos Axiom mantiene el worktree configurable y lo crea en apply" (quedará obsoleta); AGENTS.md:18-24.

## 2. Elementos abiertos: evidencia y recomendación

a) ODD en multirepo: ningún Go decide odd/tasks; ruta relativa al cwd del agente (en multirepo: maestra no-git, repo de specs o repo de código). La promoción ODD->SDD escribe proposal.md en el planning home: ya hay acoplamiento. Opciones: A) <workspaceRoot>/odd/tasks (raíz del repo con planning home); B) en cada repo de código; C) carpeta maestra (sin git). RECOMENDACIÓN A: ensure --lane odd --change <feature> aplica la política de SDD (embebido: rama+worktree antes de escribir odd/tasks; multirepo: repo de specs in situ en su rama de integración, repos de código bajo demanda con --repo porque ODD no divide en roles). Limitación: sin escritor Go de odd/tasks, A y B de D9 no cubren ODD; solo doctrina, C y D.

b) Verbo: sustantivos de primer nivel en cmd/axiom/main.go:248-468 (project, change, workspace, handoff, role, archive, review, sdd...); sdd agrupa solo verbos SDD (:2038-2060); change create ya es de primer nivel y en la práctica SDD. Hooks existentes cuelgan de su dominio (review stop-hook, sdd preflight-hook). RECOMENDACIÓN: axiom worktree ensure|status|done de primer nivel, lane-neutral (--change <nombre|feature>, --lane sdd|odd). axiom sdd worktree acopla ODD a SDD; axiom change confunde con change create. En in_place el verbo no crea worktree: imprecisión asumible. Nota: D10 fija axiom hook branch-guard; el precedente sugiere axiom worktree guard; decidir en diseño.

c) fetch: el contrato de autorización remota está inyectado en todos los runtimes (agentguidance/remote_authorization.go:10-16); la ascendencia INC-21 es deliberadamente local. Precedentes de fetch fail-open (script, dashboard) son acciones iniciadas por el usuario. RECOMENDACIÓN: sin fetch por defecto; base = origin/<integración> si existe localmente, si no la local; JSON informa base.ref/sha/source y fetched:false; --fetch explícito. status informa atraso solo con refs locales.

d) done con squash: solo existe ascendencia local (RevisionIsAncestor). Dos destinos: rol->cambio (local, fusión real, ascendencia sirve) y cambio->integración (PR remoto, a menudo squash). Detección por capas: (1) is-ancestor; (2) git merge-tree --write-tree <integración> <rama> == árbol de integración (git >= 2.38, VERIFICAR: no se pudo ejecutar git --version en esta fase); (3) evidencia pr_merged ya verificada en gates.yaml; (4) si nada, rechazar y exigir --force. [gone] del upstream requiere fetch --prune: solo informativo bajo --fetch. Siempre rechazar con árbol sucio salvo --force (D3).

e) Edición desde dashboard: no existe escritor de axiom.yaml salvo plantilla en hub/init.go:238,311-346; loader.go:64 usa yaml.Node solo para leer; declarations.go:142-182,556-584 helpers de lectura; no hay escritor que conserve comentarios (yaml.v3 conserva comentarios pero normaliza sangrías). axiom.yaml está versionado: escribirlo en el checkout principal violaría D5/D9-A. RECOMENDACIÓN: v1 solo lectura para git:/repositories:; editable únicamente lo personal de D7 (Hub). Escritor yaml.Node = seguimiento.

f) Hooks: Claude alta (PreToolUse/PostToolUse/Stop/SessionStart/UserPromptSubmit en settings global; riesgo: corre en todo repo incl. terceros). Codex media: docs oficiales (https://learn.chatgpt.com/docs/hooks) PreToolUse/PostToolUse cubren Bash y apply_patch (matchers Edit/Write), exit 2 bloquea, hooks.json ~/.codex o <repo>/.codex, activos por defecto, hooks no gestionados requieren revisión de confianza (/hooks); hoy solo SessionStart instalado. OpenCode baja-media (plugin tool.execute.before; throws pueden tragarse). Pi sin cobertura. Capa D = mejor esfuerzo; garantías = A, B, C. Debe ser casi gratuita y fail-open.

g) RDD: compatible con worktrees fuera de .claude/worktrees (1.9). Precauciones: worktree de rol y de cambio tienen linajes independientes; done debe avisar si queda linaje abierto (review abandon); slice ODD con --base-ref <rama del cambio>.

h) Dogfooding: inc-23 verificado, archivable; e2e-cumulative sumidero. Invariante D5 debe exceptuar e2e-cumulative y cumulative-<rol>. Scripts: mantener hasta publicar axiom worktree, luego envoltorios finos/retirar; pr queda fuera del verbo (D3). Diferencias: ruta (axiom-wt/<slug> vs <wt_root>/<change>[-<rol>]/<repo>), prefijos (feat|inc|fix|refactor vs un branch_prefix; hace falta --prefix/--branch), fetch por defecto. guard-primary-checkout.ps1 es política por ruta, distinta de la guarda de rama D9: actualizar mensajes a axiom worktree ensure y valorar sustituirla por axiom hook branch-guard con git.integration_branch: main. Actualizar CLAUDE.md:3,10-15, AGENTS.md:18-24, GEMINI.md.

## 3. Superficie de impacto y estimación (líneas cambiadas con pruebas)
1. D1 retirar decoupled (código, pruebas, 5 specs, docs, mensaje accionable): ~180
2. D6 esquema git:/repositories: (tipos, decodificación estricta, resolución por repo, validación): ~280
3. D2/D3/D4 motor axiom worktree ensure|status|done (planificador por topología/rol, raíz git de cada Repositories[].path, ejecutor git, registro local en <common-dir>/axiom/, JSON con rutas absolutas, ask como decision_required tipado, detección de fusión, CLI y main.go): ~1.300
4. D7 Hub (preferencias personales, registro CLI y dashboard, edición): ~450
5. D8/D9-A,B guarda común en ~8 escritores Go, blocked(wrong_branch), kickoff seal rechaza fuera de rama + instantánea git:, raíces de worktree en allowedEditRoots: ~700
6. D9-C comando CI del invariante: ~220
7. D9-D/D10 axiom hook branch-guard + instalación por sync (Claude, Codex, OpenCode): ~420
8. Dashboard/TUI ensure al crear cambios + listar cambios de worktrees registrados: ~380
9. Skills y doctrina (sdd-apply ODD-5.x, sdd-explore/archive/spec/design/propose, _shared G y status contract, routing.go + pruebas): ~380
10. Dogfooding y docs: ~180
11. Specs vivas nuevas y deltas (workspace-topology, planning-home-resolution, multi-project-hub, local-web-dashboard, sdd-lane-kickoff, sdd-archive-lifecycle, organic-agent-trigger-rules, axiom-sdd-cli-integration + nueva capacidad): ~450

TOTAL ~4.900 líneas autoradas + ~1.000 de artefactos SDD. Supera el presupuesto de 400 líneas en un orden de magnitud (single-pr elegido: solo previsión). Rebanado orientativo: S1(1+2), S2(3), S3(4), S4(5), S5(9+10), S6(8), S7(6+7).

## 4. Enfoques (bifurcaciones de implementación; motor y alcance cerrados)
- Registro local: un único registro por cambio en el common-dir del repo con planning home (rutas absolutas de todos los repos + respuesta ask); usar git rev-parse --git-common-dir, nunca .git literal.
- Instantánea en kickoff.yaml: guardar entradas de política (integración, prefijo, base por repo, nombres de rama derivados deterministas), no resultados: Seal es de una sola escritura y las ramas de rol se crean después.
- ask: resultado tipado decision_required (patrón axiom.sdd-integration.consent/v1) que presenta skill/dashboard; ensure no interactivo.
- Alcance invariante D5: solo repos con rama de cambio (ver R1).

## 5. Recomendación
Proceder a sdd-propose con D1-D10 cerradas y asunciones por defecto a confirmar con el usuario de una en una: verbo axiom worktree (b), ensure sin fetch (c), done por capas (d), dashboard solo lectura para git: (e), documento ODD en raíz del repo con planning home (a). R1-R3 y R7 son decisiones de producto reales: cerrar antes de la spec. sdd-tasks debe declarar Decision needed before apply: Yes y 400-line budget risk: High.

## 6. Riesgos y contradicciones
- R1 (D4 vs D5/D8): D4 trabaja las specs de multirepo en la rama de integración del repo de specs, pero D5 dice que esa rama solo contiene archive/ bajo openspec/changes/ y D8 hace ensure previo a toda escritura en openspec/changes/<c>/. Además git no permite dos worktrees sobre la misma rama: el repo de specs en multirepo solo puede trabajarse in situ. Fijar: invariante C y guarda wrong_branch solo para repos con rama de cambio; en multirepo ensure para specs verifica "estás en la integración y no en rama de rol".
- R2: D9-C nombra la familia axiom sdd integrate, pero esa familia es la integración en la baseline de specs de inc-24 (check|prepare|verify|show; escribe integration.yaml; cli/sdd_integrate.go:21-67). Alternativa: axiom worktree check o axiom sdd branch-check.
- R3: D5 vs sumideros e2e-cumulative y cumulative-<rol> y los dos cambios activos de main: excepción explícita y plan de transición.
- R4: workspace.Locate asciende desde cwd hasta el primer marcador (locate.go:107-133). Un worktree fuera de la carpeta maestra rompe el descubrimiento si el marcador no está dentro; deduceMaster (:454-485) solo acierta si el directorio coincide con segmentos de specs_repository; workspace validate (validator.go:107-124) busca repos de rol bajo la maestra y daría falsos errores en worktrees. Mitigación: ensure devuelve en JSON planning home y rutas; toda orden con --cwd explícito (alineado con sdd-phase-common.md G).
- R5: allowedEditRoots = [workspaceRoot]+concesiones (status.go:662,1366); worktrees de rol y de repos de código quedan fuera y darían blocked(edit_authority_missing). Proyectar en allowedEditRoots las raíces registradas por ensure y resolver rutas de tareas respecto al worktree del rol.
- R6: sdd status, dashboard y TUI listan cambios del árbol registrado; con D5 los cambios en curso viven en worktrees: agregar cambios de worktrees registrados.
- R7: D4 no define cómo vuelven las ramas de rol a la del cambio en monorepo multirrol ("un PR por repo" convive con ramas de rol; D3 excluye pr/push). Fusión local --no-ff permite ascendencia; squash no.
- R8: Repositories[].path puede ser subcarpeta de un repo, "." o no-git (validator solo IsDir): resolver raíz git (rev-parse --show-toplevel), deduplicar repos entre roles, tolerar no-git; un repo con varios roles produce varias ramas/worktrees.
- R9: estado local actual en <GitCommonDir>/gentle-ai/...; D9 fija .git/axiom/: conviven dos directorios; decidir migración o coexistencia.
- R10: capa D de mejor esfuerzo (settings Claude global; Codex exige confianza; OpenCode traga excepciones; Pi sin canal); ODD sin escritor Go: solo C y D.
- R11: fetch implícito chocaría con el contrato de autorización remota.
- R12: D1 sin alias es incompatible; el mensaje de error debe nombrar la corrección (usa multirepo).
- R13: in_place con árbol sucio: ensure debe rechazar.
- R14: Windows MAX_PATH con <wt_root>/<change>-<rol>/<repo>: validar longitud del slug.
- R15: cambios preexistentes sin instantánea git: (inc-23) exentos; kickoff seal --infer no debe exigir rama.
- R16: literales main a sustituir: kickoff/args.go:464-466, dashboard/service.go:1643 y prosa de skills.
- R17: prefijo único por proyecto vs uso mixto de este repo (inc/ feat/ fix/ refactor/): hace falta --prefix/--branch de excepción además de --base.
- R18: sin fetch la base puede estar atrasada: informar base.source/sha y avisar de atraso con refs locales.

## 7. Preparado para propuesta
Sí. Preguntas a cerrar con el usuario (una a la vez, con recomendación): (1) alcance del invariante D5 y repo de specs en multirepo (R1) + sumideros (R3); (2) integración de ramas de rol en la del cambio (R7); (3) nombre de la comprobación de CI (R2); (4) confirmar asunciones (a)-(e).
