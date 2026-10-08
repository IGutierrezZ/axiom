# Spike R17: directorio de trabajo de los ejecutores de fase (inc-24)

> Tarea 1.32 de `tasks.md` · riesgo R17 (design §10) · decisión 8 de design §13.
> Fecha: 2026-10-08 (OpenCode y primer intento de Claude Code) y 2026-10-09 (Claude Code tras renovar el login). Entorno: Windows 11, Git Bash. Solo se registran observaciones; nada se infiere ni se completa con lo que «debería» ocurrir.
> **Estado: completo.** OpenCode 1.18.27 y Claude Code 2.1.283 se ejecutaron en una sesión headless abierta en el repositorio de código.

## Escenario

Espacio canónico sintético de un solo uso (`.axiom-workspace` con `config: repo-specs/axiom.yaml`, `repo-specs/` con `axiom.yaml` y `openspec/changes/inc-99-ejemplo/tasks.md`, `repo-backend/src` y `repo-frontend`, todos con `git init`). Con el binario de la rama, `axiom sdd status inc-99-ejemplo --cwd .` desde `repo-backend` resuelve `workspaceRoot` en `repo-specs`.

Una sesión del runtime abierta en `repo-backend` (el repositorio de código) ejecuta un mensaje que le pide, paso a paso y con una llamada de herramienta por paso:

1. `pwd`.
2. Leer la ruta relativa `openspec/changes/inc-99-ejemplo/tasks.md` con shell (`cat`).
3. Leer esa misma ruta relativa con la herramienta de lectura del runtime.
4. En una sola llamada de shell: `cd "<workspaceRoot>" && pwd && cat openspec/changes/inc-99-ejemplo/tasks.md`.
5. En otra llamada: `pwd`.
6. En otra llamada que solo contiene `cd "<workspaceRoot>"`.
7. En otra llamada: `pwd`.

## OpenCode 1.18.27

Comando exacto (directorio de partida `repo-backend`, salida de eventos en JSON):

```text
opencode run "<mensaje de los siete pasos>" --dir <maestra>/repo-backend --format json --pure
```

Resultado observado, tomado de los eventos `tool_use`:

| Paso | Herramienta | Resultado |
|---|---|---|
| 1 | bash `pwd` | `<maestra>/repo-backend` |
| 2 | bash `cat openspec/changes/inc-99-ejemplo/tasks.md` | `No such file or directory` (código 1): la ruta relativa se resuelve contra el directorio de partida del runtime, que es el repositorio de código |
| 3 | read con la ruta relativa | `File not found: <maestra>/repo-backend/openspec/changes/inc-99-ejemplo/tasks.md`: la herramienta también la resuelve contra el directorio de partida |
| 4 | bash `cd "<workspaceRoot>" && pwd && cat openspec/...` | **Rechazada**: `permission requested: external_directory (<maestra>/repo-specs/*); auto-rejecting`. El runtime, en modo no interactivo, deniega por sí mismo el acceso a un directorio fuera del de la sesión |
| 5 a 7 | — | No se ejecutaron: la sesión se detuvo tras el rechazo |

Segundo experimento, para la persistencia (no depende de ningún permiso porque el destino del `cd` está dentro del directorio de la sesión). Comando: el mismo, con un mensaje de cinco pasos:

| Paso | Llamada | Salida |
|---|---|---|
| 1 | `pwd` | `<maestra>/repo-backend` |
| 2 | `cd src && pwd` (una llamada) | `<maestra>/repo-backend/src` |
| 3 | `pwd` (otra llamada) | `<maestra>/repo-backend` |
| 4 | `cd src` (otra llamada) | sin salida, código 0 |
| 5 | `pwd` (otra llamada) | `<maestra>/repo-backend` |

Conclusión para OpenCode: el `cd` de una llamada **no** persiste en la siguiente; cada llamada de shell arranca en el directorio de partida. Un `cd "<workspaceRoot>" && …` en la misma llamada es la única forma que funciona, siempre que el runtime permita acceder a `workspaceRoot`. Si queda fuera del directorio de la sesión, el runtime pide permiso `external_directory` y, sin humano, lo rechaza. No se ha comprobado qué ocurre cuando el permiso se concede, porque conceder permisos automáticamente por parámetro se consideró fuera del alcance de este experimento.

## Claude Code 2.1.283

### Primer intento (2026-10-08): sin autenticación

`claude -p` salió con código 1 sin ejecutar ninguna herramienta: `Failed to authenticate: OAuth session expired and could not be refreshed` (`apiKeySource: none`, `terminal_reason: api_error`). No se probó ninguna otra vía de autenticación. En ese momento solo se observó, en el sub-agente que ejecutaba el lote (directorio del proyecto `C:\repos\axiom`), que un `cd` de una llamada no persiste en la siguiente (`pwd` → `/c/repos/axiom`; `cd /c/repos/axiom/internal && pwd` → `/c/repos/axiom/internal`; otra llamada `pwd` → `/c/repos/axiom`).

### Segundo intento (2026-10-09): login renovado

Comprobación previa de autenticación, sin herramientas (`--disallowedTools Bash Read Edit Write`): respuesta `OK`, `is_error: false`, `terminal_reason: completed`.

Comando del escenario (directorio de partida `repo-backend`; se concede la herramienta de shell y la de lectura con `--allowedTools`, sin ningún flag que omita aprobaciones; la salida de eventos se analizó desde `stream-json`):

```text
claude -p "<mensaje de los siete pasos>" --model haiku --output-format stream-json --verbose --allowedTools Bash Read --no-session-persistence --strict-mcp-config --disable-slash-commands --setting-sources project --max-turns 20
```

Resultado observado, tomado de los eventos `tool_use` y `tool_result` (`permission_denials` vacío):

| Paso | Herramienta | Resultado |
|---|---|---|
| 1 | Bash `pwd` | `<maestra>/repo-backend` |
| 2 | Bash | El modelo sustituyó `cat` por `bat` (la configuración global del usuario lo prescribe) y falló con `command not found` (código 127); este paso no aporta evidencia y se repitió abajo con un comando literal |
| 3 | Read con la ruta relativa | `File does not exist. Note: your current working directory is <maestra>/repo-backend`: la lectura resuelve la ruta contra el directorio de partida |
| 4 | Bash `cd "<workspaceRoot>" && pwd && cat openspec/changes/inc-99-ejemplo/tasks.md` | **Ejecutada sin pedir aprobación ni rechazo**: `pwd` → `<maestra>/repo-specs` y el contenido de `tasks.md`. La salida termina con `Shell cwd was reset to <maestra>/repo-backend` |
| 5 | Bash `pwd` (otra llamada) | `<maestra>/repo-backend` |
| 6 | Bash `cd "<workspaceRoot>"` (otra llamada) | Sin salida de datos; el runtime añade `Shell cwd was reset to <maestra>/repo-backend` |
| 7 | Bash `pwd` (otra llamada) | `<maestra>/repo-backend` |

Repetición del paso 2 con un comando literal, una sola llamada de Bash: `test -f openspec/changes/inc-99-ejemplo/tasks.md; echo "exit=$?"; pwd` → `exit=1` y `<maestra>/repo-backend`: la ruta relativa no existe desde el repositorio de código.

Conclusiones para Claude Code:

- (i) Las rutas relativas `openspec/...` se resuelven contra el directorio de partida de la sesión, tanto en shell como en la herramienta de lectura, y no se encuentran desde un repositorio de código.
- (ii) `cd "<workspaceRoot>" && …` hacia un directorio hermano, fuera del de la sesión, funciona en la misma llamada con la herramienta de shell concedida por `--allowedTools`. No hubo aprobación pendiente. Con otras reglas de permisos del usuario el resultado puede diferir; no se probaron.
- (iii) El directorio no persiste entre llamadas: el propio runtime restablece el directorio de la sesión tras cada llamada y lo avisa (`Shell cwd was reset to …`).

## Conclusiones sobre la redacción de la sección G

Respaldado por observación en los dos runtimes ejecutados en una sesión abierta en un repositorio de código:

1. **Las rutas relativas `openspec/...` se resuelven contra el directorio donde se abrió el ejecutor**, no contra `workspaceRoot`. Desde un repositorio de código no se encuentran (OpenCode, pasos 2 y 3; Claude Code, pasos 2 repetido y 3).
2. **El directorio de trabajo no persiste entre llamadas de shell** (OpenCode y Claude Code). La regla pide el `cd "<workspaceRoot>" && …` en la misma llamada y nunca un `cd` previo.
3. **Acceder a `workspaceRoot` desde un repositorio de código depende del runtime**: Claude Code lo permitió con la herramienta de shell concedida, y OpenCode en modo no interactivo lo rechazó con `external_directory`. El texto de la sección G no puede asumir que el ejecutor llegará a `workspaceRoot` por sí solo.

La sección G (tarea 1.33) sigue la regla de REQ-24.11 y recoge los puntos 1 y 2, que la observación respalda en ambos runtimes; por tanto no cambia tras el segundo intento. El punto 3 queda registrado aquí como riesgo de R17 para quien abra ejecutores en un repositorio de código; la sección G no lo trata porque depende de la configuración de permisos de cada runtime y no de la convención compartida.
