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
   - [Política Git Diferencial y Gestión de Worktrees](#política-git-diferencial-y-gestión-de-worktrees-odd-5)
   - [Sincronización Continua de Especificaciones](#sincronización-continua-de-especificaciones)
6. [El Archivo de Configuración Maestro (`axiom.yaml`)](#6-el-archivo-de-configuración-maestro-axiomyaml)
   - [Política de Ramas (`git:`)](#política-de-ramas-git)
   - [Ajustes por Repositorio (`repositories:`)](#ajustes-por-repositorio-repositories)
7. [Gobernanza de Skills y Autoskills](#7-gobernanza-de-skills-y-autoskills)
   - [Flujo Human-in-the-Loop](#flujo-human-in-the-loop)
   - [Uso por CLI y por Web UI](#uso-por-cli-y-por-web-ui)
8. [Gestión de Especificaciones Vivas (OpenSpec)](#8-gestión-de-especificaciones-vivas-openspec)
   - [Podado Progresivo y Superación de Especificaciones](#podado-progresivo-y-superación-de-especificaciones-supersede)
   - [Baseline del Proyecto y Contexto Técnico](#baseline-del-proyecto-y-contexto-técnico)
9. [Flujo Dual de Desarrollo: ODD Cotidiano vs. SDD Formal](#9-flujo-dual-de-desarrollo-odd-cotidiano-vs-sdd-formal)
   - [Compuertas de Bloque e Integración Pre-Archive](#b-carril-formal-sdd-spec-driven-development)
10. [Relevos Estructurados entre Roles y Fases (Handoffs)](#10-relevos-estructurados-entre-roles-y-fases-handoffs)
11. [Mantenimiento, Actualizaciones y Sincronización](#11-mantenimiento-actualizaciones-y-sincronización)
   - [Mantenimiento Semántico del CodeGraph](#mantenimiento-semántico-del-codegraph-axiom-semantic-reindex)
   - [Limpieza Post-Merge de Worktrees](#automatización-post-archivado-y-scripts-de-soporte)

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
2. Axiom abrirá automáticamente tu navegador en el servidor local (ej. `http://localhost:4040`).
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

**Evidencia `pr_merged`.** `axiom sdd gate record --evidence-kind pr_merged` se verifica contra el grafo git del repositorio de especificaciones (`workspaceRoot`), no contra el de un repositorio de código. La integración de un repositorio de código se acredita con las evidencias `deployment` o `attestation`.

**Cambio de resolución respecto a versiones anteriores.** Antes, los verbos SDD lanzados desde un subdirectorio sin marcadores propios resolvían `<subdirectorio>/openspec`. Ahora resuelven el planning home del marcador más cercano. Lo mismo ocurre con un `axiom.yaml` directo de `multirepo` cuyo `specs_repository` no termina en `openspec`: el planning home pasa a ser `<repositorio de especificaciones>/openspec` y, si `<maestra>/openspec` existe y es otro directorio, `axiom sdd status` lo avisa en `notes`. Un binario anterior ignora el puntero en los verbos SDD, así que todo el equipo debe usar el binario nuevo.

---

### Aprovisionamiento Aislado de Agentes (`setup`)

Una vez inicializado el proyecto, ejecuta en la raíz del mismo:

```bash
axiom setup
```

**¿Qué hace `axiom setup`?**
* Ejecuta la instalación con **ámbito de workspace (`--scope=workspace`)**.
* Crea o actualiza la configuración local del proyecto (por ejemplo `opencode.json` con `"default_agent": "axiom-orchestrator"`).
* **Garantía de Aislamiento:** No altera ni sobreescribe los archivos globales de usuario en `~/.config/opencode/` ni `~/.claude/`. Cada proyecto mantiene sus agentes, plugins y prompts de forma estrictamente local.

---

### Política Git Diferencial y Gestión de Worktrees (ODD-5)

Axiom establece una separación arquitectónica estricta entre repositorios de especificaciones y repositorios de código para evitar bloqueos y colisiones:

1. **Repositorio de Especificaciones (`specs_repository` — Directo a `main`):**
   - El repositorio de especificaciones opera **siempre en la rama principal (`main` o `master`)**, sin ramas secundarias ni worktrees.
   - Las especificaciones y diseños son contratos vivos de verdad compartida. Al emitir un relevo formal (`handoff.md`), es mandatorio que las especificaciones estén commiteadas y sincronizadas con remoto (`git push`).
2. **Repositorios de Código (Worktrees y Ramas Aisladas):**
   - Antes de modificar código fuente, cada rol ejecuta el **cuestionario interactivo de pre-vuelo**:
     1. ¿Crear worktree aislado o trabajar en el árbol actual?
     2. Nombre de rama (`feat/<funcionalidad>`).
     3. Rama base (`main`, `develop`, etc.).
   - Al concluir la verificación en `PASS`, se ofrece registrar el commit y se solicita interactivamente la rama destino del Pull Request (`main`, `develop`, etc.) antes de proponer `gh pr create --base <rama-destino>`.
3. **Limpieza Automática Post-Merge (`scripts/cleanup-worktree.*`):**
   - Tras fusionar el PR, Axiom provee scripts canónicos para desmontar el worktree y eliminar la rama local de forma limpia:
     ```bash
     # Linux / macOS
     ./scripts/cleanup-worktree.sh <ruta-worktree> <nombre-rama>

     # Windows (PowerShell)
     ./scripts/cleanup-worktree.ps1 -WorktreePath <ruta-worktree> -BranchName <nombre-rama>
     ```

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

Por ahora los bloques `git:` y `repositories:` solo definen el esquema: el verbo `axiom branch` que los aplicará llega en una entrega posterior.

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

### Baseline del Proyecto y Contexto Técnico

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

### A. Carril Cotidiano: ODD (Organic-Driven Development)
Para el 90% del trabajo diario: tareas concretas, correcciones de errores y refactorizaciones sin burocracia de fases:
* **Comando:** Se gestiona mediante `axiom odd create <nombre>` o creando el documento vivo `odd/tasks/<nombre>.md`.
* **Seguimiento:** `axiom odd status` o la pestaña **ODD** en `axiom ui`.
* **Promoción:** Si una tarea ODD crece y requiere arquitectura formal, se eleva a SDD con:
  ```bash
  axiom odd promote <nombre>
  ```

### B. Carril Formal: SDD (Spec-Driven Development)
Para cambios estructurales profundos, refactorizaciones de arquitectura o requerimientos críticos:
```
[ sdd-explore ] ➔ [ sdd-propose ] ➔ [ sdd-spec ] ➔ [ sdd-design ] ➔ [ sdd-tasks ] ➔ [ sdd-apply ] ➔ [ sdd-verify ] ➔ [ sdd-archive ]
```
* **Determinación Temprana y Sello (`kickoff`):** Antes de crear la propuesta, `axiom sdd kickoff seal` fija de forma inmutable la modalidad de avance (continua o con paradas), la política de relevos y el roster de roles.
* **Compuertas de Bloque Formales:** En modalidad con paradas, cada hito exige aprobación explícita antes de avanzar:
  ```bash
  axiom sdd gate record --gate spec|design|tasks|apply --decision approved|rejected --reason "<motivo>"
  ```
* **Verificación Formal y Precondición de Integración:**
  1. `verify` valida el código mediante tests automatizados y genera `verify-report.md`.
  2. **`archive` no procede inmediatamente tras verificar:** exige registrar previamente evidencia formal de integración o despliegue en la compuerta de integración:
     ```bash
     axiom sdd gate record --gate integration --decision approved --evidence-kind pr_merged|deployment|attestation [--commit <sha>]
     ```
  3. Cumplida la compuerta, `axiom sdd archive-compose` consolida las especificaciones en `openspec/specs/`, `axiom archive sync` regenera `openspec/INDEX.md` (`archive-compose` no lo produce) y el cambio se congela en `openspec/changes/archive/`.
* **Filtro Dinámico de Roles en Web UI:** En `axiom ui` (pestaña Tareas / SDD), un selector dinámico permite filtrar las tareas activas según el rol asignado (`core`, `frontend`, `qa`, `fullstack`).

---

## 10. Relevos Estructurados entre Roles y Fases (Handoffs)

Cuando un proyecto cuenta con múltiples roles (por ejemplo `core`, `frontend`, `qa`) o transita entre fases de arquitectura e implementación, el traspaso de contexto no puede depender de mensajes informales en un chat. Axiom formaliza este paso mediante el documento canónico **`handoff.md`**.

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
En Axiom, la sincronización por defecto debe acotarse al ámbito del proyecto para aislar configuraciones y evitar colisiones:

```bash
axiom sync --scope=workspace
```

Este comando asegura que los prompts, agentes locales (`opencode.json`) y herramientas del proyecto reflejen las últimas directivas sin tocar tu configuración global de usuario en `~/.config/opencode/`.

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
