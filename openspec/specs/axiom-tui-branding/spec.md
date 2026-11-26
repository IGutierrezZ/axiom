<!-- Especificación Viva generada a partir de '2026-09-16-inc-14-axiom-tui-branding-and-cli-cutover' -->

# Especificación de Requerimientos: Arranque de axiom, Comandos de Ecosistema en CLI axiom y Pasarela de gentle-ai (INC-14)

## Propósito

Definir de forma rigurosa los requerimientos funcionales y escenarios BDD para el arranque de `axiom` (que abre la UI web desde que C-08-f retiró la TUI), el enrutamiento de comandos de instalación y sincronización en la CLI `axiom`, y la pasarela de compatibilidad de `gentle-ai`.

---

## 1. Capacidad: `axiom-tui-branding`

Arranque de `axiom` sin argumentos. La TUI que daba nombre a la capacidad se retiró en C-08-f.

## 2. Capacidad: `axiom-cli-ecosystem-commands`

Integración de la TUI y los comandos de ciclo de vida del ecosistema de agentes en la CLI soberana `axiom`.

### Requirement: Lanzamiento de TUI interactiva por defecto y subcomando axiom tui (RF-TUI-01)
Al invocar `axiom` sin argumentos, si la sesión cuenta con terminal interactivo (TTY) tanto en entrada estándar como en salida estándar, DEBE abrirse la UI web, con el mismo comportamiento que `axiom ui` (RF-WEB-08 de `local-web-dashboard`). Si no hay TTY disponible, DEBE terminar con un error y código distinto de `0` que indica que, para uso no interactivo, se usen `--version`, `axiom update` o `--help`. La opción `--help` imprime la ayuda textual. El subcomando `axiom tui` deja de existir: no queda ninguna interfaz de terminal interactiva.
(Previously: sin argumentos y con TTY se iniciaba la TUI de Bubbletea; sin TTY se imprimía la ayuda con código `0`; el subcomando `axiom tui` lanzaba siempre la TUI.)

#### Scenario: Ejecución de axiom sin argumentos en sesión interactiva TTY
- **DADO** un entorno con terminal interactivo TTY (stdin y stdout son TTY)
- **CUANDO** el usuario ejecuta `axiom` sin argumentos adicionales
- **ENTONCES** se abre la UI web igual que con `axiom ui`

#### Scenario: Ejecución de axiom sin argumentos en entorno no interactivo (CI o tubería)
- **DADO** un entorno donde stdin o stdout no son un terminal TTY
- **CUANDO** se invoca `axiom` sin argumentos
- **ENTONCES** el comando termina con un error y código distinto de `0`
- **Y** el mensaje indica que para uso no interactivo se usen `--version`, `axiom update` o `--help`

#### Scenario: El subcomando axiom tui ya no existe
- **DADO** un terminal interactivo
- **CUANDO** el usuario ejecuta `axiom tui`
- **ENTONCES** no se lanza ninguna interfaz de terminal

### Requirement: Subcomandos de gestión de herramientas y agentes en CLI axiom (RF-TUI-02)
El binario `axiom` DEBE soportar nativamente los subcomandos de gestión de herramientas: `install`, `sync`, `upgrade`, `doctor`, `backup`, `restore` y `uninstall`, delegando la ejecución a los módulos correspondientes del runtime.

#### Scenario: Ayuda de install desde axiom
- **DADO** el binario `axiom`
- **CUANDO** el usuario ejecuta `axiom install --help`
- **ENTONCES** se muestra la guía de opciones de instalación de agentes con código `0`

#### Scenario: Sincronización de agentes mediante axiom sync
- **DADO** agentes configurados en el entorno del usuario
- **CUANDO** el usuario ejecuta `axiom sync [flags]`
- **ENTONCES** se ejecutan las rutinas de sincronización de contratos y skills en los agentes soportados

#### Scenario: Diagnóstico del ecosistema mediante axiom doctor
- **DADO** un entorno de trabajo con herramientas de desarrollo
- **CUANDO** el usuario ejecuta `axiom doctor`
- **ENTONCES** se evalúa la salud de los agentes, dependencias y rutas de configuración del usuario

---

## 3. Capacidad: `gentle-ai-compat-wrapper`

Pasarela de compatibilidad y transición no disruptiva para usuarios y scripts existentes.

### Requirement: Pasarela de compatibilidad y deprecación de gentle-ai (RF-TUI-03) — RETIRADO

> **Retirado:** el binario `gentle-ai` y el paquete `cmd/gentle-ai` ya no se compilan ni se publican (ODD `seguimientos-restos-upstream`; véase RF-DIS-05 de `axiom-distribution-identity`). Se conserva el identificador RF-TUI-03 y el texto original solo por trazabilidad; el requisito no vincula.

El binario `gentle-ai` DEBE actuar como un envoltorio ligero que emite una advertencia de deprecación en `stderr` recomendando migrar al comando `axiom`, mientras delega fielmente la ejecución con todos los argumentos al motor unificado.

#### Scenario: Advertencia informativa de deprecación al invocar gentle-ai
- **DADO** una llamada al binario `gentle-ai --version`
- **CUANDO** se ejecuta el comando
- **ENTONCES** en `stderr` se emite un mensaje indicando que `gentle-ai` está deprecado y unificado en `axiom`
- **Y** en `stdout` se emite la salida esperada del comando ejecutado
- **Y** el código de salida coincide con el resultado de la operación
