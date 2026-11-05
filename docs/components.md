# Componentes, skills y presets

← [Volver al README](../README.md)

Esta guía describe los componentes que Axiom puede configurar para los agentes compatibles. Los nombres de componentes, skills y presets son identificadores estables de la CLI; un identificador heredado no implica que Gentle AI siga siendo el producto actual.

## Componentes

| Componente | ID | Descripción |
|------------|----|-------------|
| Engram™ | `engram` | Memoria persistente entre sesiones mediante MCP: detección del proyecto, búsqueda de texto, sincronización Git y consolidación de proyectos. Consulta el [repositorio de Engram](https://github.com/Gentleman-Programming/engram). Con `--scope workspace` el MCP se declara en el proyecto y `engram setup <agente>` no se ejecuta (véase [Engram y el ámbito workspace](engram.md#engram-y-el-ámbito-workspace)). |
| SDD | `sdd` | Flujo Spec-Driven Development. El agente puede usarlo cuando el trabajo lo requiera o cuando se lo pidas; no necesitas memorizar los comandos. |
| Skills | `skills` | Catálogo de skills de desarrollo que Axiom instala en la configuración de los agentes seleccionados. Con `--scope workspace` escribe una sola copia compartida en `.agents/skills` (y `.claude/skills` si Claude Code está entre los agentes); véase [Agentes](agents.md#skills-en---scope-workspace-una-sola-copia-compartida). |
| Context7 | `context7` | Servidor MCP opcional para consultar documentación de frameworks y bibliotecas. |
| Persona | `persona` | Inyección opcional de la persona de mentor (`gentleman`) o neutral; también puede mantenerse una persona personalizada sin gestión. |
| Permisos | `permissions` | Valores predeterminados y guardas de seguridad. Solo escribe reglas protectoras: la lista `permissions.deny` de Claude Code, que se une a la del usuario (sus entradas primero y en su orden, sin duplicados), y las reglas `ask` y `deny` de OpenCode y Kilo Code. Axiom nunca fija modos de autoaprobación (`bypassPermissions`, `auto_edit`, `chat.tools.autoApprove` ni un `allow` general en OpenCode y Kilo Code) ni borra los que el usuario ya tenga, de modo que Gemini CLI, Qwen Code y VS Code Copilot no reciben nada. El destino sigue el ámbito de instalación: con `--scope workspace` (el predeterminado) escribe bajo la raíz del proyecto y con `--scope global`, en el HOME. Un agente sin modo workspace se omite con un aviso y nunca se escribe en el HOME. `axiom sync` lo incluye si figura en la selección guardada o con `--include-permissions`; la TUI añade esa opción por su cuenta al descubrir Codex, sin efecto, porque Codex no tiene overlay de permisos en ningún ámbito. Las rutas sensibles de Claude Code son `.env` y `.env.*`, `**/*.pem`, `**/*.key`, `**/secrets/*` y, ancladas al HOME con `~/` porque Claude Code resuelve el resto respecto al proyecto, `~/.ssh/**`, `~/.credentials/**`, `~/Library/Keychains/**`, `~/.aws/credentials` y `~/.config/gh/hosts.yml` (las reglas antiguas sin `~/` de instalaciones previas se reescriben en su sitio y la desinstalación también las retira). OpenCode y Kilo Code deniegan la lectura de `*.env`, `*.env.*`, `**/.env`, `**/.env.*`, `**/secrets/**`, `**/credentials.json`, `**/.ssh/**`, `**/.credentials/**`, `**/Library/Keychains/**`, `**/.aws/credentials`, `**/.config/gh/hosts.yml`, `**/*.pem` y `**/*.key`. |
| GGA | `gga` | Gentleman Guardian Angel, herramienta externa para cambiar de proveedor de IA. Opcional: ningún preset la instala, hay que pedirla de forma explícita. |

### Temas visuales

Axiom no incluye temas visuales en sus presets y no los instala ni sincroniza. En `axiom sync`, la retirada se limita a los ficheros de tema que Axiom pueda reconocer como propios por ruta y contenido exactos; las preferencias existentes, los temas modificados y los ficheros de procedencia incierta se conservan. El indicador heredado `--include-theme` sigue admitiéndose al interpretar invocaciones antiguas, pero ya no instala temas. Los nombres e IDs de temas heredados no deben interpretarse como opciones de instalación vigentes.

## Autorización de operaciones remotas

Las instrucciones de agentes instaladas por Axiom incluyen un límite común para operaciones remotas, independientemente de los componentes opcionales Persona, SDD y Permisos. El acceso local de desarrollo no autoriza ejecutar operaciones remotas, transferir ficheros ni descubrir o reutilizar agentes SSH, sockets ControlMaster, credenciales o sesiones presentes en el entorno. Antes de operar en remoto se requiere autorización expresa para el destino, la operación y la credencial o sesión; la operación autorizada sigue sujeta a restricciones más estrictas del usuario o del entorno.

La primera cobertura incluye los 15 portadores principales de instrucciones distintos de Pi, no todos los roles ejecutores ni perfiles con nombre. La integración de Pi conserva su gestión en los paquetes `gentle-pi`. Estas instrucciones no proporcionan aislamiento del sistema de ficheros ni garantizan una nueva confirmación humana en cada ejecución. Por separado, los valores predeterminados de permisos de OpenCode/Kilo Code solicitan autorización para comandos `ssh`, `scp`, `sftp` y `rsync` directos, sin sustituir permisos personales. Las instalaciones existentes deben optar expresamente por sincronizar permisos; consulta el [Inicio rápido](quickstart.md). El seguimiento de las proyecciones iniciales se documentó en el [issue histórico #4324 de Gentle AI](https://github.com/Gentleman-Programming/gentle-ai/issues/4324).

## GGA

GGA es opcional y no forma parte de ningún preset, ni siquiera del predeterminado. `axiom install --component gga` instala o aprovisiona el ejecutable `gga` en el equipo; también puede elegirse en la TUI con el preset personalizado.

Las instalaciones que ya tenían GGA lo conservan: `axiom sync` sigue refrescando su configuración si figura en la selección guardada o, sin selección guardada, si Axiom ya lo aprovisionó en el HOME. Quitar GGA de los presets solo cambia lo que se instala en instalaciones nuevas.

Con `--scope workspace`, GGA sigue escribiendo en tu carpeta de usuario (su configuración en `~/.config/gga`, o en `AppData\Roaming\gga` en Windows; su runtime en `~/.local/share/gga` y, en Windows, los shims de `~/bin`, que se añade al PATH de usuario). Como es una elección explícita, se permite, pero `install` y `sync` lo declaran con un aviso que lista esas rutas, y `axiom doctor` lo muestra en su apartado de huella de máquina.

No configura automáticamente hooks para cada proyecto (`gga init` o `gga install`): esa decisión corresponde al repositorio concreto.

Después de instalar GGA, habilítalo en el proyecto en el que quieras usarlo:

```sh
gga init
gga install
```

## Skills

### Skills incluidas

Axiom incorpora skills y las inyecta en la configuración de los agentes seleccionados.

#### Spec-Driven Development

| Skill | ID | Descripción |
|-------|----|-------------|
| SDD Init | `sdd-init` | Inicializa el contexto SDD de un proyecto. |
| SDD Explore | `sdd-explore` | Investiga el código y los requisitos antes de proponer cambios. |
| SDD Research | `sdd-research` | Recopila evidencia externa cuando queda una incertidumbre concreta. |
| SDD Propose | `sdd-propose` | Define intención, alcance y enfoque de una propuesta. |
| SDD Spec | `sdd-spec` | Especifica requisitos y escenarios. |
| SDD Design | `sdd-design` | Documenta decisiones técnicas y arquitectura. |
| SDD Tasks | `sdd-tasks` | Desglosa la implementación en tareas. |
| SDD Apply | `sdd-apply` | Implementa tareas siguiendo las especificaciones y el diseño. |
| SDD Verify | `sdd-verify` | Contrasta la implementación con los requisitos y las pruebas disponibles. |
| SDD Archive | `sdd-archive` | Sincroniza las especificaciones vivas y archiva el cambio. |
| SDD Onboard | `sdd-onboard` | Guía un ciclo SDD de extremo a extremo en el repositorio. |
| Judgment Day | `judgment-day` | Revisión adversarial paralela con dos evaluadores independientes. |

#### Skills de base

| Skill | ID | Descripción |
|-------|----|-------------|
| Go Testing | `go-testing` | Patrones de pruebas Go, incluidas pruebas TUI de Bubble Tea. |
| Skill Creator | `skill-creator` | Crea skills para agentes siguiendo el formato Agent Skills. |
| Skill Improver | `skill-improver` | Audita y mejora skills frente a las convenciones del repositorio. |
| Skill Registry | `skill-registry` | Genera el índice de skills con sus activadores y rutas `SKILL.md`; cada skill apunta a una sola carpeta (la compartida `.agents/skills` si existe). Solo actúa dentro de un proyecto de Axiom (con `axiom.yaml`). |
| Chained PR | `chained-pr` | Planifica PR encadenadas revisables cuando se necesitan. |
| Cognitive Doc Design | `cognitive-doc-design` | Diseña documentación clara y fácil de revisar. |
| Work Unit Commits | `work-unit-commits` | Divide la implementación en unidades de trabajo revisables. |

Las skills de base se incluyen en los presets no mínimos. Los identificadores de skills de este repositorio que no se instalan por defecto permanecen disponibles desde el selector de la TUI y mediante la resolución explícita de `--skills`. La lista canónica se mantiene en `contributorSkills` de `internal/components/skills/presets.go`.

| Skill de contribución | ID | Descripción |
|-----------------------|----|-------------|
| Issue Creation | `issue-creation` | Plantillas para incidencias e historias. |
| Comment Writer | `comment-writer` | Borradores de comentarios de colaboración y respuesta a revisiones. |
| RDD Defect Workflow | `rdd-defect-workflow` | Tratamiento de defectos RDD con evidencia y límites de autoridad explícitos. |
| Systemic Issue Triage | `systemic-issue-triage` | Agrupa incidencias por causa sistémica. |

### Skills de programación de la comunidad

Para skills específicas de frameworks (React, Angular, TypeScript, Tailwind, Zod o Playwright, entre otros), consulta [Gentleman-Programming/Gentleman-Skills](https://github.com/Gentleman-Programming/Gentleman-Skills). Es un repositorio comunitario independiente; sus skills se instalan aparte y no son necesarias para instalar Axiom.

## Presets

| Preset | ID | Contenido |
|--------|----|-----------|
| Completo | `full-gentleman` | Engram, SDD, Skills, Context7 y Permisos; se añade la persona seleccionada salvo que se elija mantenerla personalizada. El ID se conserva por compatibilidad y no instala temas visuales. |
| Ecosistema | `ecosystem-only` | Engram, SDD, Skills y Context7; se añade la persona seleccionada salvo que se elija mantenerla personalizada. |
| Mínimo | `minimal` | Engram y las skills SDD; la persona depende de la selección. |
| Personalizado | `custom` | Permite elegir componentes y skills manualmente y dejar la persona existente sin gestionar. |

La persona se elige por separado y se aplica de forma independiente del preset. Ningún preset instala temas visuales.

GGA y CodeGraph son opcionales y ningún preset los incluye. GGA se pide con `--component gga` o con el preset personalizado; CodeGraph, una herramienta de la comunidad, se elige aparte en la TUI y nunca viene marcada por defecto. CodeGraph se instala como paquete global y escribe su MCP y su guía en la configuración de usuario de los agentes detectados, también con `--scope workspace`: Axiom no lo integra con ubicación de proyecto, así que declara esa huella igual que la de GGA.
