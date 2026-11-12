# Manual de Arranque y Operación de Axiom

> **Axiom** — Plataforma Determinista de Ingeniería de Software, Orquestación Multi-Agente y Spec-Driven Development (SDD).  
> **Destinatarios:** Equipos de ingeniería, arquitectos de software y desarrolladores que se incorporan al ecosistema.

---

## Índice

1. [Filosofía y Arquitectura](#1-filosofía-y-arquitectura)
2. [Prerrequisitos del Sistema](#2-prerrequisitos-del-sistema)
3. [Instalación en un Equipo Nuevo](#3-instalación-en-un-equipo-nuevo)
4. [Diagnóstico Inicial de Salud (`doctor`)](#4-diagnóstico-inicial-de-salud-doctor)
5. [Inicialización y Configuración de Proyectos](#5-inicialización-y-configuración-de-proyectos)
   - [Topologías Soportadas](#topologías-soportadas)
   - [Opción A: Línea de Comandos (CLI)](#opción-a-línea-de-comandos-cli)
   - [Opción B: Panel Web Interactivo (`axiom ui`)](#opción-b-panel-web-interactivo-axiom-ui)
   - [Aprovisionamiento Aislado de Agentes (`setup`)](#aprovisionamiento-aislado-de-agentes-setup)
   - [Ramas y Worktrees (`axiom branch`)](#ramas-y-worktrees-axiom-branch)
   - [Sincronización Continua de Especificaciones](#sincronización-continua-de-especificaciones)
6. [El Archivo de Configuración Maestro (`axiom.yaml`)](#6-el-archivo-de-configuración-maestro-axiomyaml)
   - [Política de Ramas (`git:`)](#política-de-ramas-git)
   - [Ajustes por Repositorio (`repositories:`)](#ajustes-por-repositorio-repositories)
   - [Carril de Trabajo (`workflow:`)](#carril-de-trabajo-workflow)
   - [Modo ISO y Perfil de Bug (`sdd:`)](#modo-iso-y-perfil-de-bug-sdd)
7. [Gobernanza de Skills y Autoskills](#7-gobernanza-de-skills-y-autoskills)
   - [Flujo Human-in-the-Loop](#flujo-human-in-the-loop)
   - [Uso por CLI y por Web UI](#uso-por-cli-y-por-web-ui)
8. [Gestión de Especificaciones Vivas (OpenSpec)](#8-gestión-de-especificaciones-vivas-openspec)
   - [Podado Progresivo y Superación de Especificaciones](#podado-progresivo-y-superación-de-especificaciones-supersede)
   - [La Spec Viva es OpenSpec](#la-spec-viva-es-openspec)
   - [Adopción de Requisitos Previos](#adopción-de-requisitos-previos-axiom-archive-adopt)
   - [Contexto Técnico](#contexto-técnico-technical_context)
   - [Baseline del Proyecto y Contexto Técnico](#baseline-del-proyecto-y-contexto-técnico)
9. [Flujo Dual de Desarrollo: ODD Cotidiano vs. SDD Formal](#9-flujo-dual-de-desarrollo-odd-cotidiano-vs-sdd-formal)
   - [Compuertas de Bloque e Integración Pre-Archive](#b-carril-formal-sdd-spec-driven-development)
10. [Relevos Estructurados entre Roles y Fases (Handoffs)](#10-relevos-estructurados-entre-roles-y-fases-handoffs)
11. [Mantenimiento, Actualizaciones y Sincronización](#11-mantenimiento-actualizaciones-y-sincronización)
   - [Mantenimiento Semántico del CodeGraph](#mantenimiento-semántico-del-codegraph-axiom-semantic-reindex)
   - [Automatización Post-Archivado y Scripts de Soporte](#automatización-post-archivado-y-scripts-de-soporte)

---

## 1. Filosofía y Arquitectura

Los agentes de inteligencia artificial para desarrollo de software sufren frecuentemente de dos limitaciones críticas:
1. **Pérdida de contexto:** Cada sesión arranca en blanco o el contexto se degrada tras compactaciones.
2. **Falta de determinismo:** Los modelos no deben "votar" qué fase sigue en un proyecto ni inventar rutas críticas.

**Axiom** resuelve estos problemas estructurando el desarrollo mediante:
* **Memoria Persistente (Engram):** Contexto acumulativo entre sesiones y agentes.
* **Gobernanza Determinista:** El estado del proyecto reside en el sistema de archivos (`axiom.yaml`, `openspec/`) y una CLI estricta en Go determina la única transición válida en cada momento.
* **Aislamiento por Workspace:** Axiom puede auto-programarse y gestionar repositorios sin sobreescribir configuraciones globales de usuario ni colisionar con otros proyectos.
* **Paridad de Interfaces:** Todo lo ejecutable por línea de comandos tiene su reflejo interactivo en la TUI de terminal (`axiom tui`) y en el Dashboard Web (`axiom ui`).

---

## 2. Prerrequisitos del Sistema

Antes de instalar Axiom en una máquina nueva, asegúrate de contar con:

* **Go 1.25+** (Axiom está compilado y validado en Go 1.25+):  
  Verifica con: `go version`
* **Git 2.38+**:  
  Verifica con: `git --version`
* **Node.js (18+) y npm** *(necesario para herramientas satélite y análisis semántico)*:  
  Verifica con: `node -v` y `npm -v`
* **Tus agentes de IA habituales** instalados en el sistema (por ejemplo OpenCode, Claude Code, Pi, Cursor, etc.).

---

## 3. Instalación en un Equipo Nuevo

Axiom se compila e instala en el `PATH` del sistema mediante la herramienta estándar de Go:

### Canal de Desarrollo Continuo (`main` — Recomendado para novedades diarias)
Permite obtener inmediatamente las últimas mejoras mergeadas sin esperar a una release empaquetada:
```bash
# Windows (PowerShell) / Linux / macOS
go install github.com/IGutierrezZ/axiom/cmd/axiom@main
```

### Canal Estable (`latest` — Hitos consolidados)
```bash
go install github.com/IGutierrezZ/axiom/cmd/axiom@latest
```

### Compilación desde el Código Fuente Local
Si has clonado el repositorio de Axiom:
```bash
cd /ruta/hacia/axiom
go install ./cmd/axiom
```

El ejecutable `axiom` (o `axiom.exe` en Windows) quedará disponible inmediatamente en tu terminal dentro de `$GOPATH/bin` (habitualmente `%USERPROFILE%\go\bin` en Windows o `~/go/bin` en Linux/macOS).

---

## 4. Diagnóstico Inicial de Salud (`doctor`)

Tras instalar, ejecuta las comprobaciones integradas:

```bash
# 1. Comprobar la versión instalada
axiom version

# 2. Diagnóstico exhaustivo del ecosistema
axiom doctor
```

`axiom doctor` audita:
* Binarios requeridos en el `PATH` (Git, Go, Node.js, npm, etc.).
* Agentes de IA detectados en el sistema.
* Disponibilidad de servidores MCP y memoria Engram.
* Integridad de herramientas semánticas (Serena, CodeGraph).
* Huella de máquina: lo que Axiom tiene a nivel de usuario (binarios, registro del hub, GGA, CodeGraph global, entradas de PATH gestionadas) y qué elementos son restos de una instalación global. Para retirarlos, `axiom uninstall --all --scope global`; la guía de migración está en [Uso: migrar de una instalación global a una por proyecto](usage.md#migrar-de-una-instalación-global-a-una-por-proyecto).

---

## 5. Inicialización y Configuración de Proyectos

Axiom puede gobernar cualquier proyecto nuevo o existente en dos topologías:

### Topologías Soportadas

1. **`monorepo-embedded` (Por defecto):**  
   El código del proyecto y las especificaciones residen en el mismo repositorio Git (bajo la carpeta `openspec/`). Ideal para la mayoría de aplicaciones, microservicios agrupados o proyectos como el propio Axiom.
2. **`multirepo`:**  
   Uno o varios repositorios de código independientes (por ejemplo `frontend/`, `backend/`, `infra/`), coordinados bajo una carpeta paraguas común, con un repositorio canónico obligatorio para las especificaciones (`specs_repository`), que incluye el caso de un único repositorio de código con las especificaciones en un repositorio aparte.

---

### Opción A: Línea de Comandos (CLI)

#### 1. Para un Monorepo
Sitúate en la raíz del repositorio y ejecuta:
```bash
axiom init --name "MiProyecto"
```
El motor de detección heurística de Axiom identificará automáticamente el lenguaje primario, los frameworks (Go, Node, React, Python, etc.) y generará:
* El contrato `axiom.yaml`.
* Las carpetas canónicas: `openspec/specs/`, `openspec/changes/` y `.axiom/inbox/skills/`.
* El registro del proyecto en el Hub global de workspaces (`~/.axiom/workspaces.json`).

#### 2. Para un Multirepo
Sitúate en la **carpeta contenedora maestra** (la carpeta que contiene los subdirectorios de cada repositorio de código y el repositorio de specs) y ejecuta:
```bash
axiom init --name "MiProyecto" --topology multirepo
```

---

### Opción B: Panel Web Interactivo (`axiom ui`)

Si prefieres una experiencia visual:

1. Lanza el dashboard desde cualquier terminal:
   ```bash
   axiom ui
   ```
2. Axiom abrirá automáticamente tu navegador en el servidor local, siempre en `http://127.0.0.1:47700` (cámbialo con `--port`). Puedes instalarlo como aplicación de escritorio: acceso directo `Axiom` del instalador, `--detach` o PWA de Edge/Chrome con el servidor arrancado por la tarea `Axiom` ([guía](usage.md#ui)).
3. En la barra superior o sección **Workspace / Proyectos**:
   * Pulsa en **"Agregar Proyecto"** o **"Inicializar Proyecto con Axiom (1 Clic)"**.
   * Introduce la ruta física (`path`), el nombre del proyecto y selecciona la topología (`monorepo-embedded` o `multirepo`).
   * El dashboard ejecutará en segundo plano el detector de tecnologías y creará la estructura sin necesidad de escribir comandos.
   * Puedes conmutar de un proyecto a otro instantáneamente desde el selector desplegable.

---

### Espacio Canónico Multirepo (Montaje Manual)

Para un multirepo con carpeta maestra no versionada, Axiom no genera la estructura: se monta a mano. La carpeta maestra contiene un puntero `.axiom-workspace` que apunta a `<specs_repository>/axiom.yaml`, el repositorio de especificaciones y los repositorios de código hermanos:

```text
<maestra>/                  carpeta maestra, no versionada
├── .axiom-workspace        puntero: config: repo-specs/axiom.yaml
├── repo-specs/             repositorio de especificaciones (git)
│   ├── axiom.yaml          topology: multirepo, specs_repository: repo-specs
│   └── openspec/           planning home: changes/ y specs/
├── repo-backend/           repositorio de código (git)
└── repo-frontend/          repositorio de código (git)
```

1. Crea la carpeta maestra y, dentro, los repositorios de especificaciones y de código.
2. Escribe `.axiom-workspace` en la carpeta maestra con `config: repo-specs/axiom.yaml`.
3. Escribe `repo-specs/axiom.yaml` con `workspace.topology: "multirepo"`, `workspace.specs_repository: "repo-specs"` y los roles de cada repositorio de código (sus rutas son relativas a la carpeta maestra).
4. Comprueba la estructura con `axiom workspace validate --path <maestra>`.

Desde la carpeta maestra, desde `repo-specs` o desde un repositorio de código (por ejemplo `repo-backend/src`), todos los verbos SDD (`sdd status`, `sdd continue`, `sdd kickoff`, `sdd gate`, `sdd-attempt grant`, `handoff` y `role`) resuelven el mismo planning home: `<maestra>/repo-specs/openspec`. El primitivo de resolución asciende desde el directorio de partida y gana el marcador más cercano (`.axiom-workspace`, `axiom.yaml` u `openspec/`). Un directorio `openspec/` propio en la carpeta maestra, junto al puntero, se rechaza por ambiguo: consolida sus cambios en el planning home o retira el puntero.

La autoridad de edición de un cambio parte del repositorio de especificaciones: los repositorios de código hermanos no se autorizan de forma implícita y requieren la autorización por cambio de `axiom sdd-attempt grant`.

Las rutas con espacios (`Repos EMT/api/x.go`) cuentan como destinos de edición en `tasks.md` solo si van ancladas: absolutas, con `./`, `../` o `~/`, con el primer segmento existente bajo la raíz del workspace, o bajo una declaración `Files:`/`Edit:`/`Touch:`. Un texto con espacios que lleva una opción (`--update`), un operador de shell (`&&`, `||`, `;`, `|`, `>`, `<`, `$(`) o una URL se sigue leyendo como comando. Las continuaciones que imprime Axiom entrecomillan esas rutas (`--cwd "C:\Repos EMT\maestra"`); el patrón `--cwd \S+` del schema publicado `contracts/sdd-integration/v1/schemas/consent.schema.json` no cubre una ruta entrecomillada con espacios, así que un validador estricto de ese schema la rechazaría (el schema está fijado por digest y no cambia).

**Evidencia `pr_merged`.** `axiom sdd gate record --evidence-kind pr_merged` se verifica contra el grafo git del repositorio de especificaciones (`workspaceRoot`), no contra el de un repositorio de código. La integración de un repositorio de código se acredita con las evidencias `deployment` o `attestation`.

**Cambio de resolución respecto a versiones anteriores.** Antes, los verbos SDD lanzados desde un subdirectorio sin marcadores propios resolvían `<subdirectorio>/openspec`. Ahora resuelven el planning home del marcador más cercano. Lo mismo ocurre con un `axiom.yaml` directo de `multirepo` cuyo `specs_repository` no termina en `openspec`: el planning home pasa a ser `<repositorio de especificaciones>/openspec` y, si `<maestra>/openspec` existe y es otro directorio, `axiom sdd status` lo avisa en `notes`. Un binario anterior ignora el puntero en los verbos SDD, así que todo el equipo debe usar el binario nuevo.

---

### Aprovisionamiento Aislado de Agentes (`setup`)

Una vez inicializado el proyecto, ejecuta en la raíz del mismo:

```bash
axiom setup
```

**¿Qué hace `axiom setup`?**
* Ejecuta la instalación con **ámbito de workspace (`--scope=workspace`)**. La raíz del workspace es la del proyecto (el `axiom.yaml` más cercano o, si no hay, la raíz del repositorio git), no la subcarpeta desde la que lances el comando; Axiom avisa cuando son distintas.
* Crea o actualiza la configuración local del proyecto (por ejemplo `opencode.json` con `"default_agent": "axiom-orchestrator"`).
* **Garantía de Aislamiento:** No altera ni sobreescribe los archivos globales de usuario en `~/.config/opencode/` ni `~/.claude/`, ni ninguna otra carpeta de configuración de usuario de los agentes. Cada proyecto mantiene sus agentes, plugins, prompts, skills y configuración MCP de forma estrictamente local, en las rutas de proyecto que lee cada agente (por ejemplo `.claude/` y `.mcp.json` en Claude Code; `opencode.json` y `.opencode/` en OpenCode; `.kilo/` en Kilo Code; `.gemini/AXIOM.md` registrado en `context.fileName` en Gemini CLI; `.codex/config.toml` y `.codex/axiom-instructions.md` en Codex; `.github/instructions/axiom.instructions.md` y `.vscode/mcp.json` en VS Code Copilot). Las skills se escriben una sola vez en `.agents/skills`, y en `.claude/skills` si Claude Code es uno de los agentes. La tabla completa está en [Agentes: modo workspace por agente](agents.md#modo-workspace-por-agente).
* **Agentes sin modo workspace:** Windsurf, Trae, Antigravity, Pi y OpenClaw se omiten con un aviso; Kimi Code, Qwen Code y Hermes ya no se pueden elegir.
* **Estado por proyecto:** lo que Axiom recuerda de la instalación (agentes, selección, persona y modelos) vive en `<raíz>/.axiom/state.json`, que no se versiona (Axiom lo excluye en `.git/info/exclude`). `~/.axiom` (o `AXIOM_STATE_DIR`) solo guarda el registro del hub, las copias de seguridad, `bin` y la caché.
* **Opcionales:** GGA y CodeGraph no se instalan por defecto; hay que pedirlos expresamente, y si los eliges conservan su huella en tu HOME, que Axiom avisa.
* **Permisos:** el componente `permissions` escribe sus reglas de denegación en el proyecto (por ejemplo `.claude/settings.json`) y nunca en tu HOME. Solo añade `deny`: no activa modos de autoaprobación como `bypassPermissions` y respeta el `deny` que ya tengas.
* **Inercia fuera del proyecto:** un directorio solo cuenta como proyecto de Axiom si tiene un `axiom.yaml` (propio o alcanzado por el puntero `.axiom-workspace`). Los hooks instalados no hacen nada en repositorios que no lo tienen, aunque lleven un `openspec/`.
* **Reversible:** `axiom uninstall --all` desde el proyecto retira lo que `setup` escribió ([Uso: uninstall](usage.md#uninstall)).

---

### Ramas y Worktrees (`axiom branch`)

Axiom prepara y retira las ramas y los worktrees de cada cambio con la familia `axiom branch`. Los orquestadores y las skills la invocan por ti (`axiom branch ensure` antes de la primera escritura de un cambio SDD o de un documento ODD); los mismos verbos están disponibles en la CLI. La política (rama de integración y prefijo) sale de `axiom.yaml` ([Política de Ramas (`git:`)](#política-de-ramas-git)); el modo de trabajo es una preferencia personal.

- `axiom branch ensure --change <cambio> [--lane sdd|odd] [--mode worktree|in_place] [--archive] [--fetch] [--json]`: crea solo lo que falta (rama del cambio, ramas de rol y sus worktrees) y devuelve las rutas absolutas con las que trabajar (`--cwd`, `git -C`). Es idempotente y no usa la red salvo con `--fetch`. Si una rama del cambio (de cambio, de rol o de archivado) no existe en local pero su referencia `refs/remotes/origin/<rama>` ya está traída (porque otra persona la publicó y hiciste `git fetch`, o con `--fetch`), `ensure` la adopta: la crea desde esa referencia con `origin/<rama>` como rama de seguimiento, sin red, e informa `createdFrom: remote` y `createdFromRef` en `branches[]` (`base.source` sigue describiendo solo la base de integración). Si no está traída y la rama se crea desde la base, el texto sugiere borrarla, hacer `git fetch origin` y repetir `ensure`; `ensure` nunca mueve una rama local existente. Con el modo `ask` y sin respuesta registrada devuelve `decision_required` (código de salida `1`) sin crear nada; se repite con `--mode`.
- `axiom branch status [--change <cambio>] [--json]`: solo lectura. Muestra el estado de las ramas y los worktrees del cambio, la rama destino propuesta para su PR (`prTarget`) y los cambios pendientes de archivar.
- `axiom branch done --change <cambio> [--force]`: tras fusionar, confirma la fusión por capas (ascendencia, árbol equivalente o PR fusionado con evidencia) y elimina el worktree y la rama local. Nunca toca una rama remota y repetirlo es idempotente.
- `axiom branch guard --agent <runtime>`: hook de la capa D (véase más abajo); no se invoca a mano.

Modelo de ramas por topología:

- **Monorepo embebido:** el cambio vive en `<prefijo>/<cambio>`, con su worktree en `<carpeta del proyecto>-wt/<cambio>/<repositorio>`. Si varios roles tocan el mismo repositorio, `<prefijo>/<cambio>` es la rama de seguimiento y cada rol saca `<prefijo>/<cambio>-<rol>` desde ella. Los artefactos de `openspec/` van en la rama del cambio desde la primera escritura.
- **Multirepo:** el repositorio de especificaciones se trabaja in situ en su rama de integración, sin ramas ni worktrees; cada repositorio de código con un rol recibe `<prefijo>/<cambio>` desde su propia rama de integración.
- La persona publica (`push`) y abre los PR; `axiom branch` nunca lo hace.
- Cada parada de un cambio SDD con paradas ofrece tres salidas (véase «Compuertas de Bloque Formales»); ODD no tiene paradas propias (véase «Carril de Trabajo (`workflow:`)»). Con las salidas 2 y 3, «aprobar, publicar y parar» y «aprobar, publicar y seguir», el orquestador encadena `axiom sdd gate record` y `axiom sdd publish`: la aprobación es la autorización explícita de subir los documentos del cambio a `origin/<rama>` con las credenciales de git ya configuradas, y de nada más; el push de código, la PR y el merge siguen siendo de la persona. Quien continúa en otra máquina no necesita una PR fusionada: en un monorepo-embedded basta `git fetch origin` (o `axiom branch ensure --fetch`, con autorización) y `axiom branch ensure --change <cambio>`, que adopta la rama publicada. En el repositorio de specs de un multirepo se publica la rama de integración, sin PR.

El modo de trabajo (`worktree` por defecto, `in_place` o `ask`) y la raíz de los worktrees no se declaran en `axiom.yaml`: son preferencias personales del hub. Se fijan al registrar el proyecto (`axiom project add --worktree-mode in_place --worktree-root <ruta absoluta> <proyecto>`, o los dos campos del modal de registro de `axiom ui`) y `axiom branch ensure --mode` las sustituye para un cambio concreto.

#### Flujo de la rama de archivado (monorepo)

1. El PR del cambio se fusiona en la rama de integración.
2. `axiom branch ensure --change <cambio> --archive` crea la rama `<prefijo>/<cambio>-archive` (y su worktree) desde la rama de integración. Si la referencia local de integración aún no contiene el cambio, rechaza con `change_not_integrated` y sugiere `--fetch`, que solo se usa con tu autorización.
3. En esa rama, `sdd-archive` registra la evidencia `pr_merged`, compone el archivado y mueve el cambio a `openspec/changes/archive/`.
4. La persona publica la rama y abre un PR documental aparte hacia la rama de integración.
5. Tras fusionarlo, `axiom branch done --change <cambio>` retira el worktree y la rama de archivado. Si el PR documental se rechaza, se descarta la rama y el cambio sigue pendiente de archivar: `axiom branch status` lo lista con la orden que reinicia el flujo.

Mientras no se archiva, un cambio ya fusionado figura como pendiente de archivar en `axiom branch status` y en el dashboard, sin bloquear nada. En multirepo no hay rama de archivado: el archive se hace in situ en el repositorio de especificaciones con la evidencia de integración o despliegue (`pr_merged`, `deployment` o `attestation`).

#### Hook de guarda (capa D) e instalación

`axiom sync` instala `axiom branch guard` como hook en Claude Code, Codex y OpenCode (no en Pi) de forma idempotente, y no instala hooks de git en ningún repositorio. El hook bloquea la escritura de documentos de Axiom (la carpeta de un cambio u `odd/tasks/<cambio>.md`) cuando el repositorio no está en la rama que corresponde, con la orden `axiom branch ensure` que lo corrige, y avisa sin bloquear en el resto de ficheros de un cambio registrado. Con el ámbito workspace, que es el predeterminado, se instala en la configuración del proyecto (`.claude/settings.json`, `.codex/hooks.json`, `.opencode/plugins/`), no en la de tu usuario. Es inerte fuera de un proyecto Axiom (el que tiene un `axiom.yaml`; un `openspec/` suelto no cuenta), falla en abierto y solo afecta a las escrituras del agente, nunca a un editor humano.

#### En este repositorio (política local)

- `axiom.yaml` declara `git: {integration_branch: main, branch_prefix: inc}`.
- El envoltorio `scripts/axiom-worktree.ps1` (o `.sh`) ofrece `new`, `pr` y `done` sobre `axiom branch`: los worktrees quedan en `C:\repos\axiom-wt\<slug>\axiom` y necesita un `axiom` con `axiom branch` en el PATH (`go install ./cmd/axiom`).
- El hook local `scripts/claude-hooks/guard-primary-checkout.ps1` deniega cualquier escritura en el checkout principal (siempre en `main`) y avisa si un comando lo deja fuera de `main`; complementa a `axiom branch guard`, que no cubre Bash ni PowerShell.

---

### Sincronización Continua de Especificaciones

Para evitar que agentes o desarrolladores trabajen sobre contratos desactualizados:
* **Detección Pasiva al Inicio de Sesión:** Al arrancar cualquier sesión, Axiom comprueba si existen commits entrantes en el repositorio de especificaciones (`behind > 0`). Si los detecta, avisa de inmediato al usuario y solicita autorización para ejecutar `git pull`.
* **Superficie Reactiva en Web UI (`axiom ui`):** Si el repositorio de specs está desincronizado con remoto, el dashboard web muestra un banner persistente en la cabecera (**"Specs Remotas Desactualizadas"**) con un botón **"Traer Cambios"** (`POST /api/workspace/specs/pull`) para sincronizar el repositorio con un solo clic.

---

## 6. El Archivo de Configuración Maestro (`axiom.yaml`)

El archivo `axiom.yaml` es la única fuente de verdad técnica de la estructura del proyecto. Ejemplo canónico:

```yaml
workspace:
  name: "MiProyecto"
  topology: "monorepo-embedded"   # Opciones: monorepo-embedded | multirepo
  specs_repository: "openspec"    # Carpeta local o subrepositorio Git de especificaciones
  skill_roots:                    # Opcional: directorios versionados con skills fuera de "skills/"
    - "internal/assets/skills"
  root: "."

roles:
  core:
    name: "Backend Core Service"
    repositories:
      - path: "."
    tech:
      - "go"
      - "postgres"
    gate_policy: "blocking"       # 'blocking' detiene el avance hasta aprobar; 'advisory' es informativo

  frontend:
    name: "Web Application"
    repositories:
      - path: "frontend"
    tech:
      - "typescript"
      - "react"
    gate_policy: "blocking"

governance:
  language: "es"                  # Idioma obligatorio de comunicación y artefactos
  shared_memory: "engram"         # Conector de memoria persistente
  semantic_analysis: "enabled"    # Serena y CodeGraph
```

`workspace.skill_roots` es opcional. Sirve para proyectos que guardan sus skills canónicas fuera de `skills/`: cada ruta debe ser relativa a la carpeta maestra y quedar dentro de ella (no se admiten rutas absolutas, `..` ni `.`). Esas raíces se escanean justo después de `skills/` y antes de los directorios de cada agente (`.claude/skills`, `.gemini/skills`, etc.), de modo que el índice de skills de `AGENTS.md` apunta a la copia versionada y no a una copia local.

Para validar en cualquier momento que la estructura de repositorios y roles cumple con las reglas:
```bash
axiom workspace validate
```

El comando también informa de cualquier entrada inválida de `workspace.skill_roots`; el escaneo de skills las ignora en silencio.

### Política de Ramas (`git:`)

`axiom.yaml` puede declarar un bloque opcional `git:` con la política de ramas del proyecto. Es política versionada que comparte todo el equipo, por eso vive en `axiom.yaml`:

```yaml
git:
  integration_branch: main   # Rama a la que se integra el trabajo (por defecto: main)
  branch_prefix: feat        # Prefijo de las ramas de cambio (por defecto: feat)
```

- Sin el bloque, o sin una de las claves, rigen `integration_branch: main` y `branch_prefix: feat`.
- Ambos valores deben ser nombres de referencia git válidos (reglas de `git check-ref-format`): cadenas no vacías, sin espacios ni caracteres de control, sin `..`, `@{`, `//` ni los caracteres `~ ^ : ? * [ \`, y sin componentes que empiecen por `.` o terminen en `.lock`. `branch_prefix` además no puede empezar ni terminar en `/` (`inc//` se rechaza).
- Una clave desconocida dentro de `git:` es un error. El modo de trabajo y la raíz de los worktrees **no** se declaran en `axiom.yaml`: son preferencias personales de cada usuario.
- `axiom workspace validate` rechaza un bloque `git:` inválido nombrando la clave, el valor y la regla incumplida, con código de salida `1`. Un `git:` inválido no impide resolver el proyecto (la resolución solo necesita la configuración base): solo fallan la carga estricta y la validación.
- Los binarios anteriores ignoran el bloque, así que declararlo no rompe a un equipo con versiones mezcladas.

### Ajustes por Repositorio (`repositories:`)

Cuando un repositorio necesita una rama de integración o un prefijo distintos de los del bloque `git:`, `axiom.yaml` puede declarar además un bloque opcional `repositories:` de primer nivel, con una entrada por repositorio. No confundirlo con la clave `repositories` de cada rol, que lista los repositorios sobre los que trabaja el rol:

```yaml
repositories:
  - path: repo-backend             # Obligatoria: ruta relativa a la carpeta maestra
    integration_branch: release    # Opcional: sustituye a git.integration_branch
  - path: repo-frontend
    branch_prefix: ui              # Opcional: sustituye a git.branch_prefix
```

- `path` es obligatoria y relativa a la carpeta maestra: no puede ser absoluta ni salir de la carpeta maestra, así que `/opt/repo`, `C:\repos\app` y `../fuera` se rechazan. Dos entradas con la misma ruta se rechazan nombrando las dos, aunque la escriban de forma distinta (`app` y `./app`).
- `integration_branch` y `branch_prefix` siguen las mismas reglas de validez que en `git:`. Un repositorio sin entrada, o una entrada sin una de las claves, hereda el valor del bloque `git:` (y, sin él, el valor por defecto): `repositories:` solo sustituye lo que declara.
- Una clave desconocida dentro de una entrada es un error, igual que una entrada sin `path`. El modo de trabajo y la raíz de los worktrees siguen sin declararse en `axiom.yaml`.
- `axiom workspace validate` rechaza un bloque `repositories:` inválido nombrando la entrada (`repositories[0]`), la clave, el valor y la regla incumplida, con código de salida `1`. Igual que `git:`, un `repositories:` inválido no impide resolver el proyecto: solo fallan la carga estricta y la validación.
- `axiom workspace validate` comprueba además en disco que cada `path` existe como directorio y obtiene su raíz git ascendiendo hasta una entrada `.git`, sin lanzar procesos (un worktree enlazado tiene su propia raíz): dos entradas que resuelven a la misma raíz git se rechazan nombrando ambas rutas y la raíz compartida (`app/web` y `app/api` dentro de un mismo repositorio). Una ruta que no pertenece a ningún repositorio git solo produce una advertencia que la nombra: el resultado sigue siendo `COMPLIANT`.
- Los binarios anteriores ignoran el bloque, como el de `git:`.

Los bloques `git:` y `repositories:` los aplica el verbo `axiom branch` (rama de integración y prefijo de cada repositorio): véase [Ramas y Worktrees (`axiom branch`)](#ramas-y-worktrees-axiom-branch).

### Carril de Trabajo (`workflow:`)

`axiom.yaml` puede declarar un bloque opcional `workflow:` con el carril por el que el agente aborda el trabajo sustancial del proyecto. Como la política de ramas, es política versionada que comparte todo el equipo:

```yaml
workflow:
  lane: ask        # odd | sdd | ask (por defecto: sdd)
```

- `odd`: carril ágil. El agente no pregunta el carril ni siquiera ante un alcance arquitectónico; solo entra en SDD si la persona lo pide de forma explícita en esa petición.
- `sdd`: carril formal. El agente entra directamente en el cuestionario de pre-vuelo de SDD, sin preguntar el carril.
- `ask`: conserva la pregunta bloqueante «¿ODD ágil o SDD formal?» en cada trabajo sustancial. Este repositorio declara `ask`.
- Sin el bloque, o sin la clave `lane`, rige `sdd`. **Cambio de comportamiento por defecto:** los proyectos con `axiom.yaml` que no declaran `workflow:` dejan de recibir la pregunta de carril y pasan a SDD; para conservar la pregunta se declara `workflow: {lane: ask}`.
- El trabajo pequeño y entendido no abre ningún carril: no crea documento de trabajo en ninguno.
- El valor distingue mayúsculas (`SDD` es inválido) y una clave desconocida es un error. `axiom workspace validate` rechaza un bloque inválido nombrando `workflow.lane`, con código de salida `1`; un `workflow:` inválido no impide planificar ramas ni resolver el espacio de trabajo.
- `axiom sdd status --json` proyecta el carril efectivo en el campo `workflow` (`lane` y `source`: `declared`, `default`, `no_workspace` o `invalid`), con o sin cambio seleccionado. Sin `axiom.yaml`, o con uno inválido, proyecta `ask`: el agente pregunta antes que entrar en SDD por su cuenta.
- ODD no tiene paradas propias: el agente crea el documento de la tarea y empieza a implementar. Las paradas con tres salidas son de las compuertas de SDD.
- Los binarios anteriores ignoran el bloque, como el de `git:`.

### Modo ISO y Perfil de Bug (`sdd:`)

`axiom.yaml` puede declarar un bloque opcional `sdd:` para un proyecto sometido a auditoría ISO. Como `workflow:`, es política versionada que comparte todo el equipo:

```yaml
sdd:
  iso: true        # true | false (por defecto: false)
```

- `iso: true` declara que el proyecto solo usa SDD: todo el trabajo, también el pequeño y entendido, se tramita por el carril SDD sin preguntar el carril (tampoco con `workflow.lane: ask`) y sin crear `odd/tasks/<tarea>.md`. La combinación con `workflow.lane: odd` es contradictoria: `axiom workspace validate` la rechaza (salida `1`, nombrando `sdd.iso` y `workflow.lane`) y `axiom sdd kickoff seal` se niega a sellar. El valor es un booleano sin comillas y una clave desconocida es un error.
- `axiom sdd kickoff seal` crea la carpeta del cambio y copia `iso` en su `kickoff.yaml`: no se crea la carpeta a mano antes. Un cambio ISO es el que se selló con esa instantánea; uno sin ella es no ISO aunque el proyecto declare ISO después, y un retrosellado (`--infer`) nunca la lleva.
- `axiom sdd status --json` proyecta el estado en el campo `iso` (`project`, `change`, `matrix` y `reason` si el bloque es `invalid`). Cada negativa es un motivo tipado `blocked(<token>)` con su continuación; la lista cerrada de trece tokens está en el contrato de status.
- Reglas de un cambio ISO: cada requisito de la delta lleva un `REQ-<n>.<m>` en su nombre y al menos un `#### Scenario:`; las tareas citan cada requisito con una línea `Cubre: REQ-x[, REQ-y ...]`; `verify` es obligatorio y escribe en `verify-report.md` la sección `## Matriz de verificación` (una fila por escenario afectado, con referencia de prueba `ruta/fichero:símbolo` o `TestX`, comando y código de salida, o responsable, fecha y justificación si es manual; nunca líneas `verdict:`); `archive` se bloquea mientras la matriz falte, sea inválida o tenga una fila `fail`. Los hallazgos de la revisión independiente siguen siendo informativos: solo bloquea la validez de la matriz.
- `axiom sdd archive-check --cwd <ruta> --change <cambio> [--json]` comprueba, sin escribir, si un cambio ISO puede archivarse; la skill `sdd-archive` lo ejecuta antes de componer o mover y se detiene si sale distinto de cero. Con `--json` informa por dominio de `alreadyComposed`: para reanudar un archivado interrumpido solo se compone lo que sigue en `false`.
- `axiom sdd trace --cwd <ruta> --change <cambio> [--integration-branch <rama>] [--json]` deriva, sin escribir, la traza del cambio desde el historial git: commit de código, commit y carpeta de archivado, y la cadena requisito, línea `Cubre:` y filas de la matriz. Una divergencia con la evidencia de la compuerta `integration` se informa como discrepancia, nunca como error.
- Los binarios anteriores ignoran el bloque, como el de `git:`.

**Perfil de bug (todo proyecto, ISO o no).** Un bug se tramita siempre por SDD, por una de tres rutas; `proposal.md` fija cuál:

| Ruta | Cuándo | Delta | `Cubre:` | Secciones de `proposal.md` |
|---|---|---|---|---|
| Requisito existente | El bug restaura un requisito que ya existe | no | cita ese requisito | `## Informe de bug` |
| Sin impacto en la spec | El bug no toca ningún requisito | no | ninguna | `## Informe de bug` y `## Impacto en la spec` |
| Con delta | El requisito estaba mal especificado o faltaba | sí | como un cambio normal | ninguna |

```markdown
## Informe de bug

Observado: el total omite el IVA reducido.
Esperado: el total aplica el IVA de cada línea.
Referencia: JIRA-1234

## Impacto en la spec

Sin impacto en la spec
Motivo: el defecto estaba en el script de despliegue, no en un requisito.
```

`Observado:` y `Esperado:` son obligatorias y `Referencia:` es opcional. La ruta sin impacto exige la línea literal `Sin impacto en la spec` y un `Motivo:` no vacío; en un cambio ISO ese motivo es la evidencia, y la matriz ausente o vacía es válida. Una declaración incompleta mantiene abierta la fase de specs con `blocked(bug_report_incomplete)` o `blocked(bug_no_impact_incomplete)`.

---

## 7. Gobernanza de Skills y Autoskills

Axiom integra un ecosistema de habilidades (*skills*) basado en el registro oficial auditado de `midudev/autoskills` (verificado criptográficamente con SHA-256) sumado a un motor de minería heurística de código local.

### Flujo Human-in-the-Loop

Axiom nunca instala una habilidad de IA a ciegas en tu proyecto. Sigue un flujo riguroso:

```
[ Escaneo / Minería ] ➔ [ Buzón Transitorio ] ➔ [ Aprobación Humana ] ➔ [ Promoción e Indexación ]
```

### Uso por CLI

1. **Escanear tecnologías y minar patrones:**
   ```bash
   axiom skill scan
   ```
   *(Las skills candidatas se depositan en el buzón transitorio `.axiom/inbox/skills/`).*

2. **Listar propuestas en el buzón:**
   ```bash
   axiom skill list --inbox
   ```

3. **Aprobar una skill para el proyecto:**
   ```bash
   axiom skill approve <nombre-de-la-skill>
   ```
   Al aprobarla, se copia formalmente a `skills/<nombre>/SKILL.md` y se regenera automáticamente el índice.

4. **Descartar una propuesta:**
   ```bash
   axiom skill reject <nombre-de-la-skill>
   ```

5. **Sincronizar el catálogo unificado en todos los destinos:**
   ```bash
   axiom skill index refresh
   ```
   Esto actualiza simultáneamente:
   * `.atl/skill-registry.md`
   * La tabla de habilidades en `AGENTS.md`
   * El tópico correspondiente en la memoria persistente de Engram MCP.

### Uso por Web UI (`axiom ui`)

1. Ve a la pestaña **Skills / Autoskills** en el panel web.
2. Pulsa en **"Escanear Skills"**.
3. Revisa la lista de habilidades propuestas, con su justificación, tecnologías asociadas y contenido.
4. Pulsa **"Aprobar"** o **"Rechazar"** directamente con un solo clic.

---

## 8. Gestión de Especificaciones Vivas (OpenSpec)

Axiom adopta y potencia el estándar **OpenSpec**:

* `openspec/specs/<dominio>/spec.md`: Especificaciones vivas consolidadas del sistema.
* `openspec/changes/<incremento>/`: Ciclos de cambio activos o en fase de propuesta.
* `openspec/changes/archive/`: Registro histórico inmutable de incrementos completados.
* `openspec/INDEX.md`: Catálogo maestro indexado de requerimientos y capacidades. Lo genera `axiom archive sync`, no `axiom sdd archive-compose`.

### Compatibilidad y Migración
* **Compatibilidad Directa:** Si un proyecto ya dispone de especificaciones en formato Markdown / OpenSpec, Axiom y su Dashboard Web las leen de forma nativa e inmediata.
* **Síntesis desde Cambios Previos (`coldstart`):** Si dispones de carpetas de cambios previos y necesitas reconstruir el catálogo de especificaciones vivas:
  ```bash
  axiom archive coldstart <nombre-cambio>
  axiom archive sync
  ```
* **Visor Web:** En `axiom ui`, la pestaña **Especificaciones** ofrece un árbol de navegación, buscador y visor con renderizado Markdown en tiempo real.

### Podado Progresivo y Superación de Especificaciones (`--supersede`)

A medida que el sistema evoluciona, nuevas capacidades reemplazan contratos previos. Para evitar que el catálogo `openspec/specs/` acumule especificaciones obsoletas o contradictorias:
* **Comando con banderas de superación:**
  ```bash
  axiom sdd archive-compose <nombre-cambio> --supersede <capacidad-obsoleta> --superseded-requirements REQ-1.1,REQ-1.2
  ```
* **Trazabilidad en el Catálogo:** El compilador de archivo retira la especificación obsoleta o genera la sección canónica `## Especificaciones Históricas Superadas`, referenciando el cambio que la sustituye y manteniendo la trazabilidad histórica completa sin contaminar los requerimientos activos.

### La Spec Viva es OpenSpec

La única spec viva de requisitos de un proyecto es `openspec/specs/` (en un `multirepo`, la del repositorio de especificaciones). Todo trabajo que cambia el comportamiento la actualiza por uno de los dos carriles, con la misma composición y la misma numeración `(REQ-n.m)`:

* **SDD:** el archivado del cambio (`sdd-archive`) integra sus deltas.
* **ODD:** el propio documento `odd/tasks/<feature>.md` lleva la delta y se integra con `axiom archive integrate --odd <feature>` (véase [Carril Cotidiano: ODD](#a-carril-cotidiano-odd-organic-driven-development)).

Un proyecto que ya tiene requisitos propios, en cualquier formato, los migra a OpenSpec con la adopción de la sección siguiente. El bloque `baseline:` de `axiom.yaml` se está retirando: la sección «Baseline del Proyecto y Contexto Técnico» describe su estado actual hasta que se retire, y la vía para dejarlo es la adopción.

### Adopción de Requisitos Previos (`axiom archive adopt`)

La adopción es un cambio SDD ordinario (kickoff, compuertas y archivado) cuya fase `spec` migra los documentos de requisitos existentes a `openspec/specs/`:

1. **Inventario.** Desde el repositorio de especificaciones y en la rama del cambio:
   ```bash
   axiom archive adopt scan --change <cambio>
   ```
   Inventaría los orígenes (`.md`, `.markdown`, `.txt`, `.docx`, `.xlsx` y `.pdf` fuera de `openspec/`), sus unidades (encabezados con su ancla), los identificadores ajenos (`RF-120`…) y los candidatos a `technical_context`. Es determinista, solo lee y, con `--change`, escribe únicamente `adoption-inventory.yaml` en la carpeta del cambio. Sin `--change` solo informa. `axiom ui` muestra el mismo inventario al inicializar el proyecto.
2. **Delta.** `sdd-spec` escribe por capacidad una delta con solo `## ADDED Requirements`, cada requisito con su `(REQ-n.m)` de una serie nueva, al menos un escenario y una línea `Origen: <ruta>#<ancla> (<ID original>)` por unidad que migra. Lo que no se migra se declara con motivo en la sección `## Descartado` de `adoption.md`. El identificador ajeno nunca es la identidad del requisito migrado.
3. **Comprobación.**
   ```bash
   axiom archive adopt check --change <cambio>
   ```
   No escribe nada ni registra compuertas: lista todos los fallos (formato, unicidad, serie, escenarios, origen, cobertura del inventario, motivo de descarte, inventario ausente o desactualizado) y sale con 1 si hay alguno.
4. **Compuerta `spec`.** Axiom comprueba la forma y la cobertura; **un humano aprueba que la migración es fiel** a sus orígenes. Ninguna orden juzga la fidelidad semántica.
5. **Archivado.** Se archiva como cualquier cambio: la spec viva conserva cada línea `Origen:` tal cual.

Los documentos de origen no se modifican. La propuesta de `technical_context` que sugiere el inventario la decide la persona.

### Contexto Técnico (`technical_context:`)

El contexto técnico es la documentación del proyecto que acompaña al código y se declara solo en `axiom.yaml` con dos modos: `maintained` (lo mantiene el archivado con un delta por cambio, `context-impact.md`) y `reference` (solo lectura: ninguna fase lo modifica). La plantilla `axiom.example.yaml` lo documenta. Con un contexto `maintained` el almacén de artefactos debe ser `openspec` o `hybrid`; `axiom workspace validate --path <maestra>` valida la declaración.

Sus comprobaciones y su integración tienen un verbo propio, en la rama de archivado para los que escriben:

```bash
axiom archive context check --change <cambio>
axiom archive context prepare --change <cambio>
axiom archive context verify --change <cambio>
axiom archive context show --change <cambio>
```

`check` valida el delta sin escribir; `prepare` fija el plan y escribe solo `integration.yaml`; `verify` comprueba la edición (traza, presencia, anclas y alcance por diff); `show` imprime la traza que se añade al informe de archivado. Un ODD comprueba además, con `axiom archive integrate --odd`, que la rama no modifique rutas de referencia.

### Baseline del Proyecto y Contexto Técnico

> **En retirada.** El bloque `baseline:` se está retirando: la única spec viva es OpenSpec. Los proyectos nuevos no lo declaran y los que lo tienen migran con `axiom archive adopt` (véase [Adopción de Requisitos Previos](#adopción-de-requisitos-previos-axiom-archive-adopt)). Lo que sigue describe su estado actual mientras no se retire.

Un proyecto que ya tiene sus propios documentos de requisitos (con su propia numeración y estructura) puede declararlos como **baseline del proyecto** en lugar de usar `openspec/specs/`. Se declara solo en `axiom.yaml` (nunca en `openspec/config.yaml`, que conserva las reglas de conducta) con dos bloques opcionales:

```yaml
baseline:
  documents:
    - domain: requisitos-funcionales        # un dominio, un documento
      path: 01_Requisitos_Funcionales.md    # relativa a workspaceRoot, dentro de la carpeta maestra
      scenarios: omit                       # include (por defecto) u omit
      requirement:                          # perfil de requisito: hace direccionable cada requisito
        heading: '^### (?P<id>RF-\d+): (?P<title>.+)$'
        level: 3
        identity: id                        # id (por defecto) o text
        render: '### {id}: {title}'
    - domain: modelo-de-datos
      path: 03_Modelo_Datos.md
      integration: directed                 # prosa: archive la edita en secciones del cambio
    - domain: flujos-de-negocio
      path: 04_Flujos_Negocio.md
      integration: register                 # prosa: archive solo la registra

technical_context:
  maintained:                               # lo mantiene el archivado con un delta por cambio
    entry: context/INDICE.md
    paths: [context]
  reference:                                # solo lectura: ninguna fase lo modifica
    paths: [referencia]
```

* **Validación.** `axiom workspace validate --path <maestra>` comprueba la declaración contra el sistema de ficheros y el almacén de artefactos. Una declaración inválida nunca degrada a la baseline por defecto: el estado de `axiom sdd status --json` la señala y bloquea solo el archivado, con la causa y esta continuación.
* **Almacén de ficheros.** Una baseline del proyecto o un contexto mantenido exigen `artifact_store` `openspec` o `hybrid` en `openspec/config.yaml`; con `engram` o `none` la validación lo reporta y el archivado se detiene.
* **Sin segunda baseline.** Con una baseline del proyecto declarada, ninguna operación de Axiom crea `openspec/specs/` ni `openspec/INDEX.md`; el catálogo y el panel informan de que la baseline es externa.
* **Integración en el archivado.** Antes de dar por terminada la fase `spec`, `axiom sdd integrate check --cwd <maestra> --change <cambio>` comprueba los deltas sin escribir. Al archivar, `axiom sdd integrate prepare` fija el plan y `axiom sdd archive-compose --cwd <maestra> --change <cambio> --document <dominio> --in-place` compone los documentos con perfil; `axiom sdd integrate verify` comprueba la edición dirigida y `axiom sdd integrate show` imprime la traza que se añade al informe de archivado.

#### Adopción en cuatro pasos

1. Monta a mano la topología canónica (receta en [Espacio Canónico Multirepo](#espacio-canónico-multirepo-montaje-manual)).
2. Declara `baseline:` y `technical_context:` en `axiom.yaml`, con `artifact_store` `openspec` o `hybrid` si se integra en ellos.
3. Ejecuta `axiom workspace validate --path <maestra>` hasta que sea conforme.
4. Para los cambios con baseline con perfil de requisito, ejecuta `axiom sdd integrate check` desde `sdd-spec`.

Un binario anterior ignora los bloques nuevos y no ve el puntero en los verbos SDD: todo el equipo debe usar el binario nuevo antes de declarar la baseline.

#### Modelo de `sdd-archive` con integración dirigida

Los documentos `integration: directed` y el contexto mantenido los edita el propio `sdd-archive`, lo que exige un modelo de nivel de implementación. Claude Code lo resuelve solo: el orquestador lanza `sdd-archive` con el modelo asignado a `sdd-apply` cuando el estado proyecta contexto mantenido o integración dirigida. En los runtimes con el modelo fijado por agente (OpenCode, Kiro y otros) no existe esa elevación: al declarar integración dirigida o contexto mantenido, asigna a `sdd-archive` un modelo de nivel de implementación con el selector de modelos existente. Nunca bloquea el archivado: `axiom sdd integrate verify` protege el resultado con cualquier modelo.

---

## 9. Flujo Dual de Desarrollo: ODD Cotidiano vs. SDD Formal

Axiom respeta la naturaleza del trabajo de ingeniería ofreciendo dos carriles complementarios:

El carril por el que el agente aborda el trabajo sustancial lo fija el bloque `workflow:` de `axiom.yaml` (`odd`, `sdd` o `ask`; por defecto `sdd`): véase [Carril de Trabajo (`workflow:`)](#carril-de-trabajo-workflow). ODD no tiene paradas propias; las paradas con tres salidas pertenecen a las compuertas de SDD.

### A. Carril Cotidiano: ODD (Organic-Driven Development)
Para el 90% del trabajo diario: tareas concretas, correcciones de errores y refactorizaciones sin burocracia de fases:
* **Documento vivo:** el agente crea `odd/tasks/<nombre>.md` y lo mantiene tarea a tarea; no existe un verbo `axiom` para crearlo, consultarlo ni promoverlo (los retiró INC-20).
* **Seguimiento:** la pestaña **ODD** en `axiom ui`.
* **Promoción:** si una tarea ODD crece y requiere arquitectura formal, se pide el carril SDD y se abre un cambio con su propuesta.
* **Spec viva (`## Spec viva`):** si el trabajo cambia el comportamiento, el documento lleva una sección `## Spec viva` con un subapartado `### Capacidad: <dominio>` por capacidad afectada y, en cada uno, un único bloque de código vallado `openspec` con la delta (`## ADDED Requirements`, `## MODIFIED Requirements`, `## REMOVED Requirements` o `## RENAMED Requirements`, con `### Requirement:` y `#### Scenario:`). Un requisito ADDED lleva su `(REQ-n.m)` en el título.
* **Integración:** en la rama de la feature, antes de cerrarla:
  ```bash
  axiom archive integrate --odd <feature>
  ```
  Valida y compone todas las capacidades antes de escribir ninguna (todo o nada), escribe solo `openspec/specs/` y, si existe, `openspec/INDEX.md`, es idempotente y no modifica el documento ODD ni crea artefactos SDD. Con `--json` emite el informe `axiom.odd-integration/v1`. Un número `(REQ-n.m)` ya usado en la spec viva o en el cambio de otra persona se rechaza con el siguiente libre.
* **Impacto de los bugs (`## Impacto en la spec`):** un documento con la línea `Tipo: bug` debe declarar su impacto: una sección `## Spec viva` con delta o una sección `## Impacto en la spec` con la línea exacta `Sin impacto en la spec` y una línea `Motivo: <una línea>`. Sin ninguna de las dos, `axiom archive integrate --odd` se niega. Sin `Tipo:` o con `Tipo: incremento` (el valor por defecto) la sección no es obligatoria.
* **Contexto técnico:** con `technical_context:` en `axiom.yaml`, la integración se niega si la rama modifica rutas del modo `reference` y te informa de las rutas `maintained` que tocó, que debes actualizar en la misma unidad de trabajo.

### B. Carril Formal: SDD (Spec-Driven Development)
Para cambios estructurales profundos, refactorizaciones de arquitectura o requerimientos críticos:
```
[ sdd-explore ] ➔ [ sdd-propose ] ➔ [ sdd-spec ] ➔ [ sdd-design ] ➔ [ sdd-tasks ] ➔ [ sdd-apply ] ➔ [ sdd-verify ] ➔ [ sdd-archive ]
```
* **Determinación Temprana y Sello (`kickoff`):** Antes de crear la propuesta, `axiom sdd kickoff seal` fija de forma inmutable la modalidad de avance (continua o con paradas) y el roster de roles. `--docs-publication auto` (solo con `--execution-style continuous`, nunca inferido) autoriza además publicar los documentos sin preguntar al cerrar `tasks` y tras archivar (`docs_publication: auto` en `kickoff.yaml`); sin él, un cambio continuo ofrece esas órdenes y espera. La política de relevos (`handoff_policy`, `--handoff-policy`) está deprecada: el flag se acepta y se ignora, y los kickoffs nuevos escriben `handoff_policy: none` solo como valor de compatibilidad con versiones anteriores. **`continuous` solo se sella con el rol único `fullstack`:** `seal --execution-style continuous` (o `--from-session-pace auto`, que deriva `continuous`) con cualquier otro roster se rechaza antes de escribir nada, y su mensaje nombra las dos salidas: sellar con `--execution-style checkpointed` o quitar las banderas `--role`. La regla es de sellado: un `kickoff.yaml` ya existente con `continuous` y varios roles se sigue leyendo, y el retro-sello `seal --infer` queda exento. Consecuencia deliberada: repetir `seal` con `continuous` y roles especializados sobre un cambio ya sellado así también se rechaza (antes respondía «Kickoff ya estaba sellado»); el kickoff en disco no se toca y se consulta con `axiom sdd kickoff show`.
* **Revisión Independiente (opcional e informativa):** Tras el pre-vuelo, el orquestador recoge en una interacción propia (no es un cuarto grupo del pre-vuelo) la preferencia de la sesión: `after_apply` (`verify`, `verify-judgment` o `none`) y `coverage` (`yes` o `no`). No se guarda en `kickoff.yaml` ni en `gates.yaml`. Con ella, al concluir el `apply` de cada rol se lanza un `verify` acotado al diff de ese rol (informe `verify-report[.<rol>].md` con la sección «Cumplimiento técnico» y la línea final `**Estado:** limpio` o `**Estado:** no limpio — <motivo>`); con `verify-judgment` y un verify limpio, un juicio de dos jueces ciegos sobre ese diff, con arreglos acotados a las raíces de edición autorizadas (`review-ledger[.<rol>].md`); y, con `coverage: yes`, una revisión de cobertura tras `spec` y tras `design` (`coverage-review.spec.md` y `coverage-review.design.md`). Ninguna bloquea una fase, una compuerta ni el archivado, no cambia `nextRecommended` ni los digests, y es ajena a RDD. En `continuous` se aplican sin detener el flujo y se resumen al final; en `checkpointed` se presentan en la parada correspondiente. Sin la preferencia (orquestador anterior), no se lanza nada por iniciativa propia en `continuous` y se ofrece la elección en la parada en `checkpointed`; una petición explícita se atiende siempre.
* **Compuertas de Bloque Formales:** En modalidad con paradas, cada hito (`spec`, `design`, `tasks` y el `role-apply` de cada rol) es una parada con tres salidas, y `axiom sdd gate record` es el único modo de registrar la decisión:
  ```bash
  # 1. Iterar: sigues trabajando sobre el artefacto y la parada se vuelve a presentar
  axiom sdd gate record --gate spec|design|tasks|role-apply:<rol> --decision rejected --reason "<motivo>"
  # 2. Aprobar, publicar y parar: cede la fase siguiente a otra persona y se publican los documentos (solo con almacén openspec o hybrid)
  axiom sdd gate record --gate spec|design|tasks|role-apply:<rol> --decision approved --hand-off [--reason "<motivo>"]
  # 3. Aprobar, publicar y seguir: se publican los documentos y el orquestador continúa con la fase siguiente (con almacén engram o none no se publica nada)
  axiom sdd gate record --gate spec|design|tasks|role-apply:<rol> --decision approved [--reason "<motivo>"]
  ```
  `gate record` solo registra: no comprueba qué está confirmado (tampoco el trabajo de un rol), no ejecuta git y, tras escribir `gates.yaml`, devuelve en `--json` la orden `publish.invocation` (`axiom sdd publish --cwd <ruta> --change <cambio> --branch <rama>`, atada a la rama con `--branch`) con el `prTarget` informativo, o `publishUnavailable` si el destino no se puede calcular. La salida 2 añade la marca de cesión y la reanudación (`git fetch origin` y `axiom branch ensure`), sin exigir una PR fusionada. La pregunta tipada de la parada (`gateQuestion`) nombra el remoto y la rama (`publication`) antes de decidir. Una publicación rechazada no deshace la aprobación ya registrada: se informa con su código y su continuación. `axiom sdd status` muestra la cesión (`handedOff`, `handedOffTo`) sin bloquear ninguna fase. La compuerta `integration` conserva sus dos salidas (aprobar con evidencia o rechazar).
* **Publicación de los documentos (`axiom sdd publish`):** `axiom sdd publish --change <nombre> [--archive] [--branch <rama>] [--json]` confirma solo las rutas de documentación del cambio (su carpeta en `openspec/changes/<nombre>/`, `openspec/specs/`, `openspec/INDEX.md` y las rutas de `baseline` y de `technical_context.maintained` declaradas en `axiom.yaml`; nunca `reference` ni `odd/tasks/`) y sube solo la rama del cambio a `origin` con `refs/heads/<rama>:refs/heads/<rama>`, sin forzar y sin saltarse hooks. El commit usa `git commit --only` sobre esas rutas: el código preparado sigue preparado y el no preparado, intacto. No hace `fetch`, no sube código ni abre PR: si la rama tiene commits de código sin subir, los informa (`code_commits_unpublished`) y los sube la persona. En el repositorio de specs de un `multirepo` sube su rama de integración; en un monorepo o en un repositorio de código nunca sube `main` (`integration_branch`). Con `--archive` publica la rama de archivado, que sin el flag es `wrong_branch`. La salida es 0 con `published` y `up_to_date`, y 1 con una negativa tipada; `--json` escribe `axiom.sdd-publish/v1` también cuando se rechaza. Solo existe con el almacén `openspec` o `hybrid` (`store_without_files` con `engram` o `none`). Lo encadena el orquestador tras cada aprobación de una parada (véase «Compuertas de Bloque Formales») y, en continuo con `docs_publication: auto`, al cerrar `tasks` y tras archivar con `--archive`. Nota para otros proyectos: el push de la rama puede disparar su CI, que queda fuera del control de Axiom.
* **Verificación Formal y Precondición de Integración:**
  1. `verify` valida el código mediante tests automatizados y genera `verify-report.md`. En un cambio ISO es obligatorio: escribe la `## Matriz de verificación` y el archivado se bloquea hasta que sea válida y con éxito (véase «Modo ISO y Perfil de Bug»).
  2. **`archive` no procede inmediatamente tras verificar:** exige registrar previamente evidencia formal de integración o despliegue en la compuerta de integración:
     ```bash
     axiom sdd gate record --gate integration --decision approved --evidence-kind pr_merged|deployment|attestation [--commit <sha>]
     ```
  3. Cumplida la compuerta, `axiom sdd archive-compose` consolida las especificaciones en `openspec/specs/`, `axiom archive sync` regenera `openspec/INDEX.md` (`archive-compose` no lo produce) y el cambio se congela en `openspec/changes/archive/`.
* **Filtro Dinámico de Roles en Web UI:** En `axiom ui` (pestaña Tareas / SDD), un selector dinámico permite filtrar las tareas activas según el rol asignado (`core`, `frontend`, `qa`, `fullstack`).

---

## 10. Relevos Estructurados entre Roles y Fases (Handoffs)

El documento **`handoff.md`** es una utilidad general de Axiom para dejar por escrito un traspaso de contexto entre roles o fases (por ejemplo `core`, `frontend`, `qa`) cuando no quieres que dependa de mensajes informales en un chat. **No es un paso del flujo SDD**: el flujo SDD no lo escribe, no lo exige y no lo consulta. La cesión de una fase a otra persona se hace con la salida 2 de las paradas (véase «Compuertas de Bloque Formales»), que la registra en `gates.yaml`, y `verify` en un cambio multirrol se habilita cuando están aprobados los `role-apply` de todos los roles `blocking` (un rol `deferred` u `optional` pendiente no lo bloquea y se nombra en una nota). Los `handoff.md` que generaron versiones anteriores siguen siendo legibles con `axiom handoff`.

### Estructura Canónica del Relevo (`handoff.md`)
Cada relevo se redacta en castellano peninsular y se divide en 5 secciones obligatorias y un Frontmatter YAML con metadatos:

```markdown
---
change: "inc-05-mi-funcionalidad"
from_phase: "design"
to_phase: "tasks"
from_role: "core"
to_role: "qa"
timestamp: 2026-09-23T20:00:00Z
status: "ready" # 'ready', 'blocked' o 'needs_clarification'
---

## 1. Resumen Ejecutivo
Síntesis clara de los objetivos completados y el estado actual del incremento.

## 2. Artefactos Modificados y Creados
Rutas exactas a los archivos de especificación, diseño, código fuente o pruebas generadas.

## 3. Decisiones Técnicas y Acuerdos
Compromisos de arquitectura adoptados, patrones empleados y dependencias establecidas.

## 4. Riesgos, Bloqueos y Preguntas Abiertas
Puntos de fricción, dependencias pendientes de terceros o dudas que requieren respuesta.

## 5. Instrucciones Directas para el Siguiente Rol
Órdenes claras, secuenciales y accionables para el agente o ingeniero que toma el relevo.
```

### Comandos de Relevo (CLI)

1. **Crear una plantilla de relevo validada:**
   ```bash
   axiom handoff create --change <slug-cambio> --from <rol-emisor> --to <rol-receptor> --from-phase <fase-origen> --to-phase <fase-destino>
   ```
2. **Consultar el relevo activo de un cambio:**
   ```bash
   axiom handoff show --change <slug-cambio>
   ```
3. **Validar la consistencia y reglas de transición:**
   ```bash
   axiom handoff validate --change <slug-cambio>
   ```

### Réplica en Memoria Persistente (Engram MCP) y Web UI
* **Espejo Automático:** Al crearse o actualizarse un relevo, Axiom lo replica en Engram bajo el tópico `sdd/<slug-cambio>/handoff`. El nuevo agente que asuma el trabajo consulta su memoria antes de re-leer ficheros innecesarios.
* **Superficie Visual:** En `axiom ui`, la pestaña **Relevos / Handoffs** permite inspeccionar gráficamente la cadena de relevos, los roles involucrados y su estado de bloqueo.

---

## 11. Mantenimiento, Actualizaciones y Sincronización

### Actualizar Axiom en tu Máquina

1. **Desde la rama `main` (desarrollo activo continuo):**
   ```bash
   go install github.com/IGutierrezZ/axiom/cmd/axiom@main
   ```
2. **Mediante el comando `upgrade`:**
   ```bash
   # Utilizando el canal beta/nightly (apuntando a main):
   axiom upgrade --channel beta

   # O canal estable:
   axiom upgrade
   ```

### Sincronizar Agentes en el Repositorio Local (Workspace-Scoped)
En Axiom, la sincronización se acota por defecto al ámbito del proyecto (`--scope workspace` es el valor predeterminado) para aislar configuraciones y evitar colisiones:

```bash
axiom sync --scope=workspace
```

Este comando asegura que los prompts, agentes locales (`opencode.json`) y herramientas del proyecto reflejen las últimas directivas sin tocar tu configuración global de usuario en `~/.config/opencode/`. Usa la raíz del proyecto aunque lo lances desde una subcarpeta y actualiza el estado de `<raíz>/.axiom/state.json`.

### Mantenimiento Semántico del CodeGraph (`axiom semantic reindex`)

Axiom integra análisis semántico del código fuente mediante CodeGraph y Serena. Tras refactorizaciones mayores o tras sincronizar cambios de otros roles:
* **Por Línea de Comandos:**
  ```bash
  axiom semantic reindex
  ```
  Actualiza incrementalmente el grafo de dependencias, tipos, funciones y llamadas del workspace sin necesidad de reiniciar la sesión.
* **Por Panel Web (`axiom ui`):**
  En la barra de herramientas superior del dashboard, el botón **"Reindexar CodeGraph"** ejecuta el proceso en segundo plano informando visualmente del progreso y completitud en tiempo real.

### Automatización Post-Archivado y Scripts de Soporte

Para acelerar el flujo tras finalizar un incremento o generar una release:

1. **Actualización local tras archivar un incremento (`scripts/post-archive-sync.ps1`):**  
   Tras mergear y archivar un cambio en `main`, este script alinea la rama `main` local, recompila el binario `axiom` con `go install ./cmd/axiom` y ejecuta `axiom sync --scope=workspace` en tu máquina para dejar el entorno 100% operativo al instante.
2. **Lanzamiento de Release Oficial (`scripts/release-milestone.ps1`):**  
   Cuando un incremento represente un hito mayor consolidado, se ejecuta pasando el número de versión (ej. `./scripts/release-milestone.ps1 -Version v0.2.0`). El script valida que el repositorio esté limpio, genera el tag anotado en Git y hace `push` hacia GitHub, disparando el flujo automatizado de GoReleaser sin intervención manual.

---

*Axiom — Rigor de Arquitectura, Determinismo y Memoria para Agentes de Software.*
