# Rediseño visual del dashboard de Axiom

Propuesta de rediseño de `internal/dashboard/assets` (`index.html`, `style.css`, `app.js`). La guía está pensada para quien vaya a implementarla, sea una persona o Claude Code. El objetivo es que el dashboard resulte cómodo durante toda la jornada a todo el equipo (desarrollo, PM, PO, análisis y QA). Para ello añade un tema claro, oscuro o automático, guardado por usuario. También sustituye las 7 pestañas por una navegación lateral y crea una pantalla de Inicio que dice qué necesita atención hoy.

> **Importante:** los mockups usan **datos inventados** (proyecto `portal-clientes`, incrementos como `checkout-v2`). Son una referencia visual y de comportamiento. No son código para copiar tal cual. Todos los datos reales salen de la API existente (`/api/*`). Los huecos de la API se listan en [Huecos de datos](#huecos-de-datos).

## Ruta rápida

1. Lee esta guía completa y abre los mockups (ver [Cómo ver los mockups](#cómo-ver-los-mockups)).
2. Aplica `tokens.css` como base de `style.css`. Implementa después las unidades de trabajo en el [orden propuesto](#orden-de-implementación).
3. Verifica cada unidad con `go test ./internal/dashboard/...` y una revisión visual en claro y oscuro, a 1440 px y a 390 px.

## Contenido de esta carpeta

| Ruta | Qué es |
|------|--------|
| `README.md` | Esta guía: decisiones, mapa pantalla→API, huecos y orden de trabajo |
| `tokens.css` | Variables de color, tipografía y radios para claro y oscuro, listas para usar |
| `mockups/canvas.json` | Índice del lienzo: lista de pantallas y su disposición |
| `mockups/Main.dc.html` | Inicio |
| `mockups/Incrementos.dc.html` | Tablero de incrementos (tablero por fase y lista) |
| `mockups/Detalle.dc.html` | Detalle de incremento, pestaña Artefactos |
| `mockups/Roles.dc.html` | Detalle de incremento, pestaña Roles y barrera |
| `mockups/Handoffs.dc.html` | Detalle de incremento, pestaña Handoffs |
| `mockups/Skills.dc.html` | Skills y buzón de propuestas |
| `mockups/Specs.dc.html` | Specs vivas |
| `mockups/Codigo.dc.html` | Código y grafo (semántica) |
| `mockups/Ecosistema.dc.html` | Ecosistema: Doctor, operaciones, topología, modelos, respaldos |

### Cómo leer un `.dc.html`

Los mockups usan el formato de la herramienta de diseño. **No se pueden servir tal cual**:

| En el mockup | Significa | En Axiom |
|--------------|-----------|----------|
| `<x-dc>`, `<helmet>`, `./support.js` | Envoltorio de la herramienta de diseño | Descartar |
| `{{nombre}}` | Valor calculado | Valor de la API pintado desde `app.js` |
| `<sc-for list="{{xs}}">` | Repetición | Bucle en `app.js` que genera el DOM |
| `<sc-if value="{{x}}">` | Condicional | Mostrar u ocultar según los datos |
| `class Component extends DCLogic` | Lógica de demostración con datos falsos | **No copiar.** Solo sirve como ejemplo del comportamiento esperado (filtros, aprobar o rechazar, tema) |
| `href="Detalle.dc.html"` | Enlace entre pantallas | Ruta interna, por ejemplo `#/incrementos/<name>` |
| `style="…"` en línea | Valores exactos de diseño | Trasladar a clases en `style.css` con los tokens |

Lo valioso de cada mockup es la estructura HTML semántica (`<nav>`, `<section>`, `<button>`, `<table>`, `aria-*`), los tamaños, espaciados y radios exactos, y los textos en castellano.

## Decisiones de diseño

| Tema | Decisión |
|------|----------|
| Navegación | Barra lateral de 248 px con tres grupos: **Trabajo** (Inicio, Incrementos), **Conocimiento** (Specs vivas, Skills, Código y grafo) y **Plataforma** (Ecosistema). En pantallas estrechas se apila arriba (`flex-wrap`); no es fija ni usa `100vh` |
| Selector de proyecto | Bloque en la parte superior de la barra con iniciales, nombre y «topología · N roles». Abre la lista de `/api/projects` y las acciones Explorar y Añadir del header actual |
| Barra superior | Ruta de navegación (breadcrumb), búsqueda global (Ctrl K: incrementos, specs, skills y símbolos), botón Actualizar (solo icono, con `aria-label`) y botón principal **Nuevo incremento** |
| Roles y Handoffs | **Dejan de ser pestañas globales.** Pasan a ser pestañas dentro del detalle de cada incremento (Artefactos · Roles y barrera · Handoffs). Desaparecen los desplegables «Seleccionar cambio…» |
| Inicio (nueva) | Titular accionable («N incrementos necesitan atención»), aviso de sincronización de specs, 4 KPI, lista «Necesita atención» con la acción ya preparada, «Flujo SDD» por fase, «Axiom te ayuda» y actividad reciente |
| Tema | Claro, Oscuro o Auto (sigue al sistema). Selector en el pie de la barra lateral, con el texto «Tema · se guarda para ti» |
| Tipografía | **Instrument Sans** para la interfaz y **JetBrains Mono** para identificadores, fases, rutas y código. Texto base de 14 px; títulos de página de 30 px con peso 650 |
| Color | Base neutra cálida y un único acento azul. Estados: ok (verde), aviso (ámbar) y error (rojo). Los bugs se marcan siempre en rojo (`--bad`) |
| Iconos | SVG en línea, trazo `currentColor`, `viewBox="0 0 24 24"` y grosor 1.8. Sin emojis. Sustituyen al mapa `ICONS` de `app.js` |
| Accesibilidad | `<button>` y `<a href>` reales, `aria-pressed` en controles conmutables, `aria-current` en la navegación, `aria-label` en botones de solo icono y contraste ≥ 4.5:1 en ambos temas. Los estados no se distinguen solo por el tono |

### Correspondencia con las pestañas actuales

| Pestaña actual (`data-tab`) | Pantalla nueva |
|-----------------------------|----------------|
| `tab-increments` | Incrementos (`Incrementos.dc.html`) y detalle (`Detalle.dc.html`) |
| `tab-roles` | Detalle → Roles y barrera (`Roles.dc.html`) |
| `tab-handoffs` | Detalle → Handoffs (`Handoffs.dc.html`) |
| `tab-skills` | Skills (`Skills.dc.html`) |
| `tab-semantic` | Código y grafo (`Codigo.dc.html`) |
| `tab-livingdoc` | Specs vivas (`Specs.dc.html`) |
| `tab-ecosystem` | Ecosistema (`Ecosistema.dc.html`) |
| — | **Inicio** (`Main.dc.html`), pantalla nueva |
| `#zero-config-hero` | Se mantiene: sustituye el contenido de Inicio cuando `is_configured` es `false` |

### Tema: implementación

El dashboard se ejecuta en local, así que basta con guardarlo en el navegador de cada usuario.

1. Clave `localStorage`: `axiom.theme`, con valores `light`, `dark` o `system`. Valor por defecto: `system`.
2. Para evitar el parpadeo inicial, añade un `<script>` en línea en el `<head>`, antes de `style.css`. Debe leer la clave y fijar `document.documentElement.dataset.theme` a `light` o `dark` (con `system`, el valor se resuelve mediante `matchMedia('(prefers-color-scheme: dark)')`).
3. Con `system`, escucha el evento `change` de ese `matchMedia` para seguir al sistema operativo en vivo.
4. Envuelve en `try/catch` cada lectura y escritura de `localStorage`. Si falla, usa `system`.
5. Sustituye `class="modern-light-theme"` del `<body>` por el atributo `data-theme` en `<html>`.

### Fuentes sin conexión

`assets.go` embebe `assets/*` en el binario, y el dashboard debe funcionar sin red. **No cargues Google Fonts.** Copia los `.woff2` de Instrument Sans (400, 500, 600 y 700) y JetBrains Mono (400, 500 y 600) en `assets/fonts/` y decláralos con `@font-face`. Comprueba antes que `//go:embed assets/*` incluye la subcarpeta. Si no la incluye, cambia la directiva a `assets` o `assets/**`. Las pilas de `tokens.css` ya tienen respaldos del sistema.

## Mapa pantalla → API

Todas las rutas existen hoy en `server.go`. Los DTO están en `types.go`.

### Inicio (`Main.dc.html`)

| Bloque | Fuente | Regla |
|--------|--------|-------|
| KPI «Incrementos activos» | `GET /api/increments` | `type == "active"`; los bugs son los que tienen `is_bug` |
| KPI «Tareas completadas» | ídem | Suma de `tasks_completed` / `tasks_total` de los activos |
| KPI «En verificación» | ídem | `phase == "verify"`; subtítulo con el número de `ready_for_archive` |
| KPI «Salud del ecosistema» | `GET /api/ecosystem/doctor` | `healthy` y número de comprobaciones con `status != "ok"` |
| Aviso de specs | `GET /api/workspace/specs/sync-status` | Visible si `behind > 0`; el botón llama a `POST /api/workspace/specs/pull` |
| Necesita atención | `GET /api/increments` | Prioridad: `ready_for_global_verify` → **Verificar** (`POST /api/increments/verify`); `waiting_roles` → **Ver barrera**; `ready_for_archive` → **Archivar** (ver huecos); `pending_spec` y `ready_for_design` → **Continuar** (`POST /api/increments/continue`) |
| Flujo SDD | `GET /api/increments` | Conteo por `phase` (explore…verify) más los archivados; las barras rojas corresponden a `is_bug` |
| Axiom te ayuda | `/api/skills/inbox`, `/api/semantic/status`, `/api/archive/specs`, `/api/ecosystem/doctor` | Una fila por fuente; se oculta si no hay nada relevante |
| Actividad reciente | — | **Hueco** (ver abajo) |
| «Ver como» (rol) | `GET /api/workspace` (`roles`) | Filtra «Necesita atención» por rol; ver huecos |

### Incrementos (`Incrementos.dc.html`)

| Bloque | Fuente | Regla |
|--------|--------|-------|
| Columnas | `GET /api/increments` | Por `phase`: explore, spec, design, tasks, apply, verify y archive (los archivados aparecen atenuados, con el enlace «Ver los N archivados») |
| Chip de tipo | `change_type`, `is_bug` | feature → Feature (acento); fix o `is_bug` → Bug (rojo); refactor → neutro; architecture → Arquitectura (ámbar) |
| Progreso | `tasks_completed`, `tasks_total`, `progress_pct` | La barra se pone verde al llegar al 100 % |
| Bandera de estado | `pending_spec`, `ready_for_design`, `waiting_roles` (+ `pending_roles`), `ready_for_global_verify`, `ready_for_archive` | Como máximo una bandera por tarjeta, en ese orden de prioridad inverso |
| Filtros | Los mismos de hoy (`data-filter`) | Son chips con contador; filtro de rol a partir de `/api/workspace` |
| Vista Lista | ídem | Tabla con un minipaso de 7 fases por fila |
| Chips de rol | — | **Hueco**: el resumen solo trae `pending_roles` |

### Detalle de incremento (`Detalle.dc.html`, `Roles.dc.html`, `Handoffs.dc.html`)

| Bloque | Fuente | Regla |
|--------|--------|-------|
| Cabecera y paso a paso de fases | `GET /api/increments/<name>` | `has_proposal`, `has_spec`, `has_design`, `has_tasks`, `has_verify` y `has_archive` marcan cada paso; la fase actual es `summary.phase` |
| Descripción | `proposal` | Primer párrafo de la propuesta |
| Lista de artefactos y visor | `proposal`, `spec`, `design`, `tasks_content`, `verify_report`, `archive_report` | El visor tiene un conmutador entre la vista renderizada y el Markdown en bruto |
| `tasks.md` por rol | `tasks_content` | Agrupar por los encabezados de rol y pintar `- [x]` / `- [ ]` como iconos |
| Siguiente paso | Derivado | Texto según banderas y barrera; botón **Continuar con Axiom** → `POST /api/increments/continue` |
| Verificar | `POST /api/increments/verify` | Queda deshabilitado (`aria-disabled` con `title`) mientras la barrera esté bloqueada |
| Roles y barrera | `GET /api/roles?change=<name>` | `satisfied`, `blockers` y `roles[*].role.{name,gate_policy,repositories,tech}`, `progress`, `verify_verdict`. El rol único `fullstack` muestra el aviso de flujo continuo actual |
| Migrar rol diferido | `POST /api/increments/migrate-cumulative` `{change, role}` | Solo en roles con `gate_policy == "deferred"` |
| Handoff | `GET /api/handoffs?change=<name>` | Secciones: resumen ejecutivo, artefactos, decisiones, riesgos e instrucciones |
| Crear handoff | `POST /api/handoffs` (`CreateHandoffRequest`) | Reutiliza el modal actual con el nuevo estilo |
| Specs afectadas | — | **Hueco** |

### Skills (`Skills.dc.html`)

| Bloque | Fuente |
|--------|--------|
| Buzón (lista y vista previa) | `GET /api/skills/inbox` (`SkillProposalDTO`: `origin`, `verified`, `role`, `detected_by`, `justification`, `created_at`, `skill_md`) |
| Aprobar / Rechazar | `POST /api/skills/approve` y `POST /api/skills/reject` (`SkillActionDTO`) |
| Escanear | `POST /api/skills/scan` |
| Skills activas | `GET /api/skills` (`name`, `description`, `trigger`); filtro en cliente |

### Specs vivas (`Specs.dc.html`)

| Bloque | Fuente |
|--------|--------|
| Lista de dominios y métricas | `GET /api/archive/specs` |
| Visor del dominio | `GET /api/archive/specs/<domain>`. Renderizar los requisitos y los escenarios con DADO/CUANDO/ENTONCES |
| Sincronizar catálogo | `POST /api/archive/sync` |
| Historial y cambios en curso | **Hueco** |

### Código y grafo (`Codigo.dc.html`)

| Bloque | Fuente |
|--------|--------|
| Estado de los conectores | `GET /api/semantic/status` (`active_connector`, `codegraph_installed` y `_configured`, `serena_installed` y `_configured`) |
| Agentes y MCPs | La misma fuente que la lista de agentes actual (`#semantic-agents-list`) |
| Símbolos | `GET /api/semantic/symbols`, con filtros de tipo struct, interface, func y method |
| Dependencias | `GET /api/semantic/dependencies` |
| Reindexar | `POST /api/semantic/reindex` |

### Ecosistema (`Ecosistema.dc.html`)

| Bloque | Fuente |
|--------|--------|
| Banner de salud y Doctor | `GET /api/ecosystem/doctor`; las comprobaciones con aviso o error van primero |
| Operaciones | `POST /api/ecosystem/sync`, `POST /api/ecosystem/upgrade` `{channel}` y `POST /api/ecosystem/backups/create` |
| Última operación | Respuesta de la operación (`EcosystemActionResponse`, con `phases` cuando hay `upgrade → sync`) en un `<details>` |
| Topología | `GET /api/workspace` (`name`, `topology`, `specs_repository`, `roles`) |
| Modelos | `GET /api/ecosystem/models` (`active_persona`, `assignments`) |
| Respaldos | `GET /api/ecosystem/backups`; restaurar con `POST /api/ecosystem/backups/restore` |

## Huecos de datos

Hoy no los cubre la API. **Decide con el usuario** si se implementan o si se omite el bloque. No inventes datos en la interfaz.

| Hueco | Pantalla | Propuesta mínima |
|-------|----------|------------------|
| Actividad reciente | Inicio | Endpoint `GET /api/activity` construido con las fechas de handoffs, archivados y aprobaciones de skills, o bien omitir la tarjeta |
| Estado por rol en el resumen | Incrementos, Inicio («Ver como») | Añadir a `IncrementSummaryDTO` una lista de roles con su estado, o pintar solo `pending_roles` |
| Archivar desde el dashboard | Inicio, Detalle | No hay endpoint para archivar un incremento. Mientras no exista, la acción enlaza al detalle con la orden equivalente |
| Varios handoffs por incremento | Handoffs | La API devuelve uno. Mientras tanto, la lista lateral muestra solo ese |
| Specs afectadas por un incremento | Detalle, Specs vivas | Exponer los deltas de `openspec/changes/<name>/specs/*` |
| Historial de versiones de un dominio | Specs vivas | Omitir hasta que exista una fuente |
| Nombres de modelo | Ecosistema | En el mockup aparecen como `[modelo]`; se usan los reales de `/api/ecosystem/models` |

## Orden de implementación

Cada unidad cabe en un commit revisable (de unas 400 líneas como referencia orientativa) con sus pruebas.

| # | Unidad | Resultado comprobable |
|---|--------|-----------------------|
| 1 | Tokens y tema | `tokens.css` integrado, script anti-parpadeo y selector Claro/Oscuro/Auto que persiste. La estructura no cambia |
| 2 | Estructura y navegación | Barra lateral, barra superior y rutas por hash (`#/inicio`, `#/incrementos`, `#/incrementos/<name>/roles`…) que sustituyen a `.nav-tab` |
| 3 | Incrementos | Tablero por fase, vista Lista y filtros con contadores |
| 4 | Detalle de incremento | Pestañas Artefactos, Roles y barrera, y Handoffs, más el modal de crear handoff |
| 5 | Inicio | KPI, Necesita atención, Flujo SDD y Axiom te ayuda (sin bloques con hueco salvo decisión expresa) |
| 6 | Skills y Specs vivas | Buzón con vista previa y visor de requisitos |
| 7 | Código y Ecosistema | Conectores, símbolos, dependencias, Doctor, operaciones, modelos y respaldos |

Notas para quien implemente:

- Mantén la arquitectura actual: HTML, CSS y JS sin framework, embebidos con `go:embed`. No añadas dependencias de build.
- Conserva los `id` que usen `app.js` o los tests, o actualízalos de forma coherente. Revisa `dashboard_test.go` antes de renombrar.
- El rediseño no cambia el contrato de la API. Los huecos se tratan como tareas aparte.
- Todo el texto de la interfaz va en castellano, igual que en los mockups.

## Checklist de aceptación

- [ ] Las 7 vistas actuales siguen accesibles con la nueva navegación y no se pierde ninguna acción existente.
- [ ] El tema se guarda por usuario, sobrevive a la recarga y Auto sigue al sistema en vivo.
- [ ] No hay parpadeo de tema al cargar.
- [ ] Contraste ≥ 4.5:1 en texto y etiquetas en ambos temas.
- [ ] Todo es usable a 390 px: la barra lateral se apila, la barra superior se ajusta y las tablas anchas se desplazan dentro de su caja.
- [ ] Todos los controles son accesibles con teclado y tienen foco visible (`outline` de 2 px en el color de acento).
- [ ] Ningún dato de los mockups aparece en producción.
- [ ] Las fuentes se sirven embebidas y funcionan sin red.
- [ ] `go test ./internal/dashboard/...` pasa.

## Fuera de alcance

- Nuevos endpoints o cambios de DTO, salvo los que se acuerden en [Huecos de datos](#huecos-de-datos).
- Cambios de comportamiento del flujo SDD u ODD.
- Persistencia del tema en `~/.axiom` (posible mejora posterior si se quiere compartir entre navegadores).

## Cómo ver los mockups

- **Lienzo original:** el artefacto de diseño en claude.ai (privado del autor; pedir acceso). Ahí se pueden ejecutar en modo Play: tema, filtros, aprobar o rechazar skills y búsqueda de símbolos.
- **En local:** los `.dc.html` no se renderizan solos. Léelos como especificación de estructura y estilos. Las capturas de referencia deben tomarse del lienzo.
