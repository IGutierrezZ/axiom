# ODD: rediseño visual y de navegación del dashboard web

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/dashboard-redesign.md`, en el worktree `C:\repos\axiom-wt\dashboard-redesign` (rama `feat/dashboard-redesign`).
> Espejo de recuperación en Engram: topic `odd/dashboard-redesign/tasks`, proyecto `axiom`.
> **Estado: En curso (2026-10-29).** Carril ODD elegido por el usuario.
> Referencia de diseño: `docs/redesign/` (README, `tokens.css` y 9 mockups `.dc.html` de Claude Design). **Es material de referencia, no especificación**: lo que manda es este documento.

## Objetivo

Sustituir la interfaz de `internal/dashboard/assets` (`index.html`, `style.css` y `app.js`) por el nuevo diseño. Debe incluir:

- tema claro, oscuro o automático guardado por usuario;
- barra lateral con tres grupos, barra superior y rutas por hash;
- pantalla de Inicio que diga qué necesita atención;
- Roles y Handoffs dentro del detalle de cada incremento.

Debe seguir siendo HTML, CSS y JS sin framework, embebido con `go:embed` y usable sin red.

## Problema y porqué

- El dashboard actual tiene 7 pestañas planas y un solo tema. No hay vista que resuma qué requiere acción, y Roles y Handoffs obligan a elegir el cambio en un desplegable aparte.
- `/api/roles` y `/api/handoffs` serializan estructuras sin etiquetas `json`:
  - `multirole.BarrierReport` (`internal/multirole/types.go:75`) y `handoff.Handoff` (`internal/handoff/types.go:60`) salen con claves Go (`Satisfied`, `Metadata.FromPhase`…);
  - `app.js` lee `satisfied` y `metadata.from_phase`;
  - hoy Roles muestra siempre el aviso «fullstack» y Handoffs lanza `TypeError`.
- Lo que dice `docs/redesign/README.md` se revisó contra el código (2026-10-29) y tiene errores. Los relevantes:
  - `assets/**` no es sintaxis válida de `go:embed`, y `assets/*` ya embebe subcarpetas;
  - afirma que la API no cambia, pero el contrato de roles está roto;
  - el contraste de `--text-3` sobre `--hover` en claro es 4,33:1, por debajo del 4,5:1 que promete;
  - los mockups quitan el `outline` de foco en las búsquedas y distinguen estados solo por color;
  - el tablero de 7 columnas esconde Verify y Archive a 1440 px;
  - no hay ninguna regla para 390 px.

## Decisiones del usuario (2026-10-29)

- **D1 · El README es orientativo.** El diseño visual de los mockups es la referencia; las reglas de datos y comportamiento se deciden aquí.
- **D2 · Los tests se rediseñan.** Pueden cambiar o romperse y se reescriben con la interfaz, como una refactorización. Los tests que comprueban subcadenas de `app.js`, `index.html` y `style.css` (`fs_test.go:127-164`, `dashboard_test.go:164-186` y `:984-1015`, `archive_baseline_test.go:170-190`) se sustituyen por comprobaciones del comportamiento nuevo.
- **D3 · Huecos de datos.** Se aceptan las recomendaciones:

  | Hueco | Decisión |
  |---|---|
  | Actividad reciente | Se omite |
  | Estado de cada rol en las tarjetas | Solo `pending_roles`; la barrera completa, en el detalle |
  | Varios handoffs | Se muestra el único que hay |
  | Specs afectadas | Se omite |
  | Historial de dominio | Se omite |
  | Fechas relativas | Solo las fechas existentes (`date`, `last_modified`, `Timestamp`) |

- **D4 · Handoffs, esfuerzo mínimo.** Se eliminarán en una versión próxima. Se arregla su JSON para que no falle, se muestra en el detalle con el estilo nuevo y el modal de crear se conserva reestilizado. No se añade nada nuevo.
- **D5 · Política de acciones.**
  - **Botón:** solo las acciones que el servidor ya ejecuta como orden de Axiom o de git.
  - **Recordatorio:** lo que exige una fase de agente. Es un texto, no un botón, y dice qué paso toca y que se lance desde el IDE o CLI favorito, con la orden copiable.

  | Acción | Tratamiento | Fuente |
  |---|---|---|
  | Continuar | Botón; salida en consola | `POST /api/increments/continue` → `cli.RunSDDContinue` |
  | Nuevo incremento | Botón y modal | `POST /api/increments` |
  | Migrar rol diferido | Botón | `POST /api/increments/migrate-cumulative` |
  | Escanear, aprobar o rechazar skill | Botón, con confirmación al aprobar o rechazar | `/api/skills/{scan,approve,reject}` |
  | Reindexar | Botón | `POST /api/semantic/reindex` |
  | Sincronizar specs vivas | Botón | `POST /api/archive/sync` |
  | Pull de specs | Botón | `POST /api/workspace/specs/pull` |
  | Sync, upgrade y crear respaldo | Botón | `/api/ecosystem/*` |
  | Restaurar respaldo | Botón, con confirmación | `POST /api/ecosystem/backups/restore` |
  | Crear handoff | Botón y modal (D4) | `POST /api/handoffs` |
  | Verificar | **Recordatorio** («lanza la fase verify desde tu IDE o CLI») | Fase de agente. Hoy el endpoint solo comprueba que existe `verify-report.md` |
  | Archivar | **Recordatorio** | No hay endpoint; es la fase de agente archive |
  | Propuesta, spec, diseño, tareas y apply | **Recordatorio** con la fase siguiente | Fases de agente |

- **D6 · Entrega.** El usuario concede excepción de tamaño (`exception-ok`). Las tareas se agrupan por coherencia, no por líneas. Habrá un PR con un commit (o más) por tarea.

## Alcance autorizado

- `internal/dashboard/assets/**`: `index.html`, `style.css`, `app.js` (o varios JS, todos embebidos), `fonts/` e iconos.
- `internal/dashboard/types.go`, `service.go`, `server.go` y `assets.go`, solo para:
  - los DTO de roles y handoff (T1);
  - el tipo MIME `.woff2` (T2);
  - el arnés de vista previa (T2).
- `internal/dashboard/*_test.go`, que se rediseñan (D2).
- `docs/redesign/**`, que es referencia y se commitea tal cual.
- **Fuera de alcance:**
  - endpoints nuevos (actividad, archivar, specs afectadas, historial);
  - cambios del flujo SDD u ODD;
  - persistir el tema en `~/.axiom`;
  - la TUI (`internal/tui`);
  - retirar el endpoint `/api/increments/verify`, que queda sin uso en la interfaz y se anota como seguimiento.

## Restricciones

- Commits en castellano, Conventional Commits, **sin `Co-Authored-By` ni atribución de IA**.
- Worktree `C:\repos\axiom-wt\dashboard-redesign`; `C:\repos\axiom` queda en `main`, sin ediciones.
- **EDR (Cortex XDR):** prohibido `go build -o` y `go run ./cmd/axiom`. Solo `go test`, `go build ./...`, `go vet` y `go run ./internal/gofmtcheck`. La vista previa en navegador se hace con un arnés dentro de `go test` (T2).
- Sin dependencias externas en tiempo de ejecución. Nada de Google Fonts ni CDN: las fuentes `.woff2` van embebidas con su licencia OFL.
- No se copia la lógica `DCLogic` ni ningún dato inventado de los mockups (`portal-clientes`, `checkout-v2`…).
- Accesibilidad y 390 px se cumplen **en cada tarea**, no al final:
  - foco visible de 2 px en acento;
  - `aria-current`, `aria-pressed` y `aria-label` en los botones de solo icono;
  - estados con texto o icono además del color;
  - contraste de al menos 4,5:1 en ambos temas.
- Push, PR y merge, solo con permiso del usuario.

## Modo de trabajo

- **TDD:** estándar. Fuente: `openspec/config.yaml` (`strict_tdd: false`). Runner: `go test ./internal/dashboard/...`. Las reglas derivadas que se extraigan a Go, si se hace, llevan su test.
- **RDD:** apagado (`axiom review mode status` → `off (decided by default)`, 2026-10-29). Sin revisión nativa; se aplican las comprobaciones ordinarias.
- **Verificación visual:** en el panel de navegador, con el arnés de T2, en claro y oscuro, a 1440 px y a 390 px.
- **Entrega:** `exception-ok` (D6), un solo PR.

## Tareas

Ruta de cada tarea: «delegada» significa un escritor acotado, que se elige porque toca dos o más ficheros no triviales.

- [x] **T0 · Referencia de diseño y documento ODD** (en línea)
  - Commit de `docs/redesign/` tal cual y de este documento.
  - Comprobación: `git status` limpio tras el commit.
  - Evidencia: commit `6eb44edc`.
- [x] **T1 · Contrato JSON de roles y handoff** (delegada: `types.go`, `service.go` y test)
  - Crear DTOs propios en `dashboard`, con `snake_case`, para `GET /api/roles`, `IncrementDetailDTO.barrier_report` y `GET /api/handoffs`. Los tipos de `multirole` y `handoff` no se tocan.
  - Roles: `satisfied`, `blockers`, `warnings` y `roles[]` con `{name, gate_policy, repositories, tasks{total,completed,pending,percent}, verify_done, verdict, compliant}`, más `deferred_tasks[]`.
  - Handoff: `metadata{change, from_phase, to_phase, from_role, to_role, timestamp, status}` y `sections{…}`.
  - Test de contrato que serialice una barrera y un handoff y compruebe las claves.
  - El `app.js` actual se adapta lo mínimo para que no falle hasta T3.
  - Comprobación: `go test ./internal/dashboard/...`.
  - Resultado (2026-10-29):
    - `contract_dto.go` con `NewBarrierDTO` y `NewHandoffDTO` en el límite del dashboard, y DTO en `types.go`. `multirole` y `handoff` no cambian.
    - Cada rol expone `role` (id) y `name` (nombre visible, opcional), además de `gate_policy`, `repositories`, `deliverables`, `tasks_found`, `tasks{total,completed,pending,percent}`, `apply_done`, `verify_done`, `verdict` y `compliant`.
    - `timestamp` va en RFC 3339, o `""` si es cero.
    - `app.js` adaptado: `roles` como array y `sections.direct_instructions`. Antes leía `next_instructions`, que no existía.
  - Verificación:
    - `go build ./...`, OK;
    - `go vet ./internal/dashboard/...`, OK;
    - `go test ./internal/dashboard/... ./internal/tui/...`, ok (repetido por el orquestador);
    - `gofmt -l internal/dashboard`, limpio.
  - Sin probar en navegador; se cubre en T3.
- [ ] **T2 · Base visual y estructura** (delegada)
  - **Tokens y tema:**
    - `tokens.css` integrado en `style.css`, con `--text-3` corregido para llegar a 4,5:1 sobre `--hover`;
    - script anti-parpadeo en `<head>` y `data-theme` en `<html>`;
    - selector Claro, Oscuro o Auto en el pie de la barra lateral;
    - `localStorage` `axiom.theme` con `try/catch`, y Auto que siga a `matchMedia` en vivo.
  - **Fuentes:**
    - Instrument Sans (400, 500, 600 y 700) y JetBrains Mono (400, 500 y 600) en `assets/fonts/`, con `OFL.txt`;
    - `@font-face`;
    - `mime.AddExtensionType(".woff2", "font/woff2")`.
  - **Estructura:**
    - barra lateral de 248 px con grupos Trabajo, Conocimiento y Plataforma, apilada en estrecho con `@media`;
    - barra superior con miga, Actualizar y Nuevo incremento;
    - rutas por hash (`#/inicio`, `#/incrementos`, `#/incrementos/<name>/{artefactos,roles,handoff}`, `#/specs`, `#/skills`, `#/codigo`, `#/ecosistema`) que sustituyen a `.nav-tab`.
  - **Selector de proyecto** en la barra lateral, como popover accesible con `aria-expanded`. Reutiliza `/api/projects*` y el selector de carpetas existente, con el estilo nuevo.
  - **Componentes comunes:**
    - botones, chips, tarjetas y tablas con desplazamiento interno;
    - iconos SVG en línea que sustituyen al mapa `ICONS`;
    - modal y consola de ejecución;
    - estados vacío, carga y error;
    - componente «Recordatorio» de D5 con orden copiable.
  - **`app.js` en módulos por pantalla:** varios `<script>` clásicos o `type="module"`, sin paso de compilación, con un helper `api()` común para `fetch` y errores.
  - **Zero-config:** `#zero-config-hero` sustituye a Inicio cuando `is_configured` es `false`.
  - **Arnés de vista previa:** test que solo se activa con una variable de entorno y levanta el servidor real en un puerto fijo dentro de `go test`. Sirve para revisar en el navegador sin compilar `axiom.exe`.
  - **Tests rediseñados:** sirve `/`, `/style.css`, el JS y `/fonts/*.woff2` con su `Content-Type`.
  - Las pantallas aún sin migrar se muestran temporalmente con su contenido antiguo bajo la estructura nueva.
- [ ] **T3 · Flujo de trabajo: Incrementos y detalle** (delegada)
  - **Tablero por fase:** las columnas vacías se pliegan y las de fases con acción (verify) quedan siempre visibles; se resuelve el desbordamiento a 1440 px.
  - **Vista lista:** con minipaso de 7 fases.
  - **Filtros:** chips con contador y filtro por rol a partir de `/api/workspace`.
  - **Tarjetas:**
    - chip de tipo, siempre con texto;
    - progreso, que se pone verde al llegar al 100 %;
    - una sola bandera de estado;
    - `pending_roles`.
  - **Detalle con pestañas:**
    - Artefactos: lista, visor renderizado o en bruto y `tasks.md` agrupado por rol;
    - Roles y barrera, con los DTO de T1, aviso fullstack, bloqueos y Migrar en los roles `deferred`;
    - Handoff, según D4.
  - «Siguiente paso»: Continuar es un botón; Verificar y Archivar son recordatorios (D5).
  - Modal Nuevo incremento y consola de Continuar.
  - Desaparecen las pestañas globales Roles y Handoffs y sus desplegables.
- [ ] **T4 · Inicio** (delegada)
  - Titular accionable. Cuenta los incrementos con alguna bandera de atención; la regla se define en el código y en este documento.
  - Aviso de specs por detrás con botón Pull.
  - 4 KPI: activos y bugs, tareas, en verificación y salud del ecosistema.
  - «Necesita atención», con prioridad `ready_for_global_verify` > `waiting_roles` > `ready_for_archive` > `pending_spec`/`ready_for_design`, y acción según D5.
  - Flujo SDD por fase, con los bugs marcados con texto además de color.
  - «Axiom te ayuda»: buzón de skills, estado semántico, specs y doctor; se oculta si no hay nada relevante.
  - Insignias de la barra lateral (incrementos, skills y aviso de ecosistema) con datos cacheados y refrescados por Actualizar o el sondeo existente de 60 s, no en cada cambio de ruta.
  - Sin actividad reciente ni «Ver como» (D3).
- [ ] **T5 · Conocimiento: Skills, Specs vivas y Código** (delegada)
  - **Skills:**
    - buzón con lista y vista previa, mostrando `source`, `origin`, `verified`, `role`, `detected_by`, `justification` y `skill_md`;
    - Aprobar y Rechazar con confirmación;
    - Escanear;
    - skills activas con filtro.
  - **Specs vivas:**
    - dominios con filtro y métricas;
    - visor de requisitos y escenarios DADO/CUANDO/ENTONCES a partir del markdown;
    - Sincronizar;
    - aviso de baseline externa (`catalog.external_baseline`).
  - **Código y grafo:**
    - conectores, incluido `native_ast_ready`;
    - agentes y MCPs;
    - símbolos con filtros de tipo usando los parámetros de servidor (`query`, `kind`);
    - dependencias;
    - Reindexar.
- [ ] **T6 · Plataforma: Ecosistema** (delegada)
  - Banner de salud y Doctor, con avisos y errores primero y mapa de traducción `warning`/categorías → castellano.
  - Operaciones Sync, Upgrade con canal y Crear respaldo.
  - Última operación en `<details>`, con las fases `upgrade → sync` y los literales de estado actuales (`restart-required`, `upgrade-failed`…).
  - Topología desde `/api/workspace`.
  - Modelos, `agent` × `role`, desde `/api/ecosystem/models`.
  - Respaldos con tamaño a partir de `size_bytes` y Restaurar con confirmación.
- [ ] **T7 · Limpieza y verificación final** (en línea y en el navegador)
  - Eliminar el CSS y el JS muertos de la interfaz antigua.
  - Recorrer en el navegador las 7 vistas antiguas y comprobar que no se pierde ninguna acción existente salvo las convertidas en recordatorio por D5.
  - Revisar en claro y oscuro, a 1440 px y a 390 px.
  - Navegación completa con teclado.
  - Contraste medido de los pares de tokens.
  - Ningún dato de los mockups en producción.
  - `go test ./internal/dashboard/...`, `go vet ./internal/dashboard/...` y `go run ./internal/gofmtcheck`.

## Criterios de aceptación

- Las 7 vistas actuales son accesibles con la nueva navegación. Toda acción existente sigue disponible como botón o, si es una fase de agente, como recordatorio (D5).
- Roles y barrera muestran los datos reales del JSON corregido (T1).
- El tema persiste por usuario, no parpadea al cargar y Auto sigue al sistema en vivo.
- Funciona sin red: fuentes e iconos embebidos.
- Es usable a 390 px y con teclado, con contraste de al menos 4,5:1 en ambos temas.
- `go test ./internal/dashboard/...` pasa con los tests rediseñados.

## Progreso y evidencias

- 2026-10-29: análisis del README y de los mockups contra `server.go`, los DTO y los tests (dos exploraciones delegadas y comprobación directa de `multirole/types.go:75`, `handoff/types.go:60` y `service.go:531,553`). Decisiones D1–D6 registradas.

## Siguiente paso

T2: base visual y estructura.

## Seguimientos

- Retirar o redefinir `POST /api/increments/verify`, que queda sin uso en la interfaz (D5).
- Retirar Handoffs del dashboard y de la API (D4, decisión del usuario para una versión próxima).
- Endpoints de actividad, archivar, specs afectadas e historial de dominio, si se quieren recuperar esos bloques (D3).
