# Inicio rápido

Axiom es el producto de este repositorio. Esta guía cubre la instalación con un solo comando, la configuración de los agentes que ya utilizas y las comprobaciones iniciales. La integración de Axiom no instala los runtimes de los agentes ni aplica temas visuales.

## Requisitos

- Linux y macOS: solo `curl`; el instalador descarga el archivo firmado de la última release. No hacen falta Go ni Git.
- Windows: Go 1.25.10 o posterior (el binario oficial está retenido hasta la firma Authenticode, ver [`release-signing.md`](release-signing.md)). Si falta, `winget install --id GoLang.Go -e` y abre una terminal nueva.
- Go 1.25.10 o posterior y Git 2.38 o posterior solo si vas a contribuir o compilar desde el checkout (sección «Contribuir / desarrollo»).
- Node.js 18 y npm. Axiom los comprueba y muestra una advertencia con una sugerencia si faltan; no los instala. Son necesarios para CodeGraph y para las integraciones que usan paquetes npm.
- Pi disponible como `pi` en `PATH` si vas a seleccionar esa integración.
- Axiom admite macOS, Linux y Windows. En Linux, la compatibilidad de instalación se limita a Ubuntu/Debian, Arch y Fedora/RHEL y sus derivados indicados por el diagnóstico.

## Instalar Axiom

Un solo comando, sin clonar el repositorio.

Linux y macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/IGutierrezZ/axiom/main/scripts/install.sh | bash
axiom version
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/IGutierrezZ/axiom/main/scripts/install.ps1 | iex
axiom version
```

El script de Windows ejecuta `go install github.com/IGutierrezZ/axiom/v3/cmd/axiom@latest`; el ejecutable queda en el directorio `bin` de Go (`GOBIN` o, si no está definido, `GOPATH/bin`), que debe estar en `PATH`. Para instalar desde `main` (canal beta): en Linux y macOS, `curl -fsSL https://raw.githubusercontent.com/IGutierrezZ/axiom/main/scripts/install.sh | AXIOM_CHANNEL=beta bash` (la variable debe aplicarse a `bash`, no a `curl`); en PowerShell, `$env:AXIOM_CHANNEL = "beta"` antes del comando `irm ... | iex`.

## Contribuir / desarrollo

Solo si vas a modificar Axiom: clona el repositorio y compila el ejecutable `axiom`.

```sh
git clone https://github.com/IGutierrezZ/axiom.git
cd axiom
go install ./cmd/axiom
axiom version
```

Para probar la versión del checkout sin instalarla:

```sh
go run ./cmd/axiom version
```

## Primera configuración

Empieza validando el plan sin aplicar cambios:

```sh
axiom install --dry-run
```

Si el plan es correcto, aplica la configuración de los agentes y componentes seleccionados:

```sh
axiom install
```

Por defecto, `axiom install` escribe en el proyecto (`--scope workspace`) y no en tu carpeta de usuario; para instalar en la configuración global de los agentes hay que pedir `--scope global`. Detalle en [Uso](usage.md#install).

Axiom configura los agentes detectados o seleccionados. Algunas integraciones admiten aprovisionar su runtime, mientras que otras requieren instalación manual; los detalles por agente figuran en [Agentes compatibles](agents.md). La selección de agentes no instala ni sincroniza temas visuales.

Los agentes seleccionados durante la instalación se convierten en el ámbito predeterminado de `axiom sync`. Para comprobar los cambios previstos antes de sincronizar:

```sh
axiom sync --dry-run
```

Para especificar agentes concretos, indica cada destino:

```sh
axiom sync --agent claude-code --agent opencode
```

## Verificación y solución de problemas

Ejecuta el diagnóstico de solo lectura para comprobar la instalación y las herramientas detectadas:

```sh
axiom doctor
```

Si falla una comprobación de instalación o sincronización, revisa primero el resultado del diagnóstico y el informe de verificación. La guía de [copias de seguridad y restauración](rollback.md) explica cómo volver a una configuración respaldada sin confundir una copia de Axiom con una copia histórica de Gentle AI.

Para abrir la interfaz de terminal de forma explícita:

```sh
axiom tui
```

El comando `axiom` también abre la TUI cuando se ejecuta en una terminal interactiva. `axiom ui` inicia el panel web local en `http://127.0.0.1:47700`; en Windows el instalador añade el acceso directo `Axiom` al menú Inicio y una tarea programada que arranca el servidor al iniciar sesión (véase [uso](usage.md#ui)).

## Protecciones integradas

Al seleccionar `permissions`, Axiom aplica a Claude Code, OpenCode y Kilo Code una lista de rutas sensibles que incluye `~/.ssh/*`, `**/*.pem`, `**/*.key`, `**/.env*` y `~/.aws/credentials`. Solo añade reglas de denegación y de confirmación: no activa ningún modo de autoaprobación y, con `--scope workspace`, escribe en el proyecto y no en tu HOME. Consulta el detalle en [Componentes](components.md).

## Plataformas no compatibles

Si la plataforma o distribución no está soportada, Axiom detiene la instalación y muestra el sistema detectado. No uses opciones de otra plataforma para forzarla: consulta primero la salida de `axiom doctor` y la documentación específica del sistema.
