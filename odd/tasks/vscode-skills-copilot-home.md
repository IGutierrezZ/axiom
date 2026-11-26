# ODD: skills globales de VS Code Copilot con `COPILOT_HOME` (vscode-skills-copilot-home)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/vscode-skills-copilot-home.md`. Espejo Engram: `odd/vscode-skills-copilot-home/tasks`.
> Rama: `inc/vscode-skills-copilot-home` · worktree `C:\repos\axiom-wt\vscode-skills-copilot-home\axiom`.

## Objetivo

Decidir, con la documentación oficial, si las skills globales de VS Code Copilot (y sus agentes de usuario) deben ir a `$COPILOT_HOME/skills` cuando `COPILOT_HOME` está definida, como ya hacen el MCP global de VS Code (IGutierrezZ/axiom#325) y todo lo de Copilot CLI. Después, aplicar la decisión en el código o en la documentación.

## Problema y porqué

`vscode.Adapter.SkillsDir` devuelve siempre `<home>/.copilot/skills`, y `GlobalConfigDir` devuelve `<home>/.copilot`. Copilot CLI y el MCP global de VS Code usan `copilotconfig.Root(homeDir)`. Esa función respeta `COPILOT_HOME` solo cuando `homeDir` es el home real. Con `COPILOT_HOME` definida, VS Code deja las skills en `~/.copilot/skills` y Copilot CLI las deja en `$COPILOT_HOME/skills`. La IGutierrezZ/axiom#325 dejó esta divergencia fuera de alcance.

## Evidencia de la documentación oficial (consultada el 26/11/2026)

- https://code.visualstudio.com/docs/copilot/customization/agent-skills (redirige a `/docs/agent-customization/agent-skills`):
  - La tabla de ubicaciones dice, para el arnés Copilot: «Project: `.github/skills/` Personal: `~/.copilot/skills/`».
  - No menciona `COPILOT_HOME` en ningún punto de la página.
  - Dice que las skills son portables entre VS Code y Copilot CLI por ser un estándar abierto, pero no habla de una carpeta compartida.
- https://code.visualstudio.com/docs/agents/reference/custom-agent-file-reference:
  - Ubicación de usuario: «User (Copilot) | `~/.copilot/agents` folder».
  - No menciona `COPILOT_HOME`.
  - Su tabla de *placeholders* tiene una fila «Copilot configuration directory», pero el nombre del *placeholder* sale vacío en el HTML publicado. No prueba nada sobre `COPILOT_HOME`.
- https://code.visualstudio.com/docs/agents/reference/mcp-configuration es la única página de VS Code que nombra `COPILOT_HOME`, y siempre para MCP:
  - Destino de usuario: «`$COPILOT_HOME/mcp-config.json`, or `~/.copilot/mcp-config.json` when `COPILOT_HOME` is unset».
  - En el descubrimiento de configuraciones MCP de otras aplicaciones: «For GitHub Copilot CLI, setting `COPILOT_HOME` replaces the default `~/.copilot` location».
- https://code.visualstudio.com/docs/agents/run/agent-harnesses y https://code.visualstudio.com/docs/agent-customization/overview: no mencionan `COPILOT_HOME` ni `~/.copilot`.
- https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference describe solo Copilot CLI:
  - `agents/` y `skills/` viven dentro del directorio de configuración.
  - Cita literal: «`COPILOT_HOME` replaces the entire `~/.copilot` path».
  - No dice nada del arnés de VS Code.

## Decisiones

- **No se migra: la documentación no lo confirma.** VS Code documenta `COPILOT_HOME` solo para `mcp-config.json`. Para las skills y los agentes de usuario documenta `~/.copilot/skills` y `~/.copilot/agents` sin condiciones. Que el arnés de VS Code comparta motor con Copilot CLI y por eso respete `COPILOT_HOME` también para las skills es una inferencia sin respaldo en el texto. Sin evidencia, se queda la ruta documentada: si se migrase y VS Code no leyera `$COPILOT_HOME/skills`, quien use VS Code con `COPILOT_HOME` perdería las skills.
- **`SkillsDir`, `GlobalConfigDir` y `SubAgentsDir` de VS Code no cambian.** El comentario del adaptador explica por qué no usan `copilotconfig.Root`, con las URL.
- **Desinstalación compartida (`skillsDirStaysInUse`):** no cambia. Compara las rutas resueltas. Sin `COPILOT_HOME`, la carpeta es la misma y protege a la del otro agente. Con `COPILOT_HOME`, son dos carpetas y cada agente retira solo las suyas. Lo fija `TestUninstallCopilotCLIWithCopilotHomeRemovesOnlyItsOwnSkills`.
- **Detección (`copilotconfig.HasCLIState`):** no cambia. `skills/` y `agents/` siguen sin contar como señal del CLI. Con `COPILOT_HOME`, `$COPILOT_HOME/skills` solo lo escribe Copilot CLI, pero no contarlo es conservador: el CLI se sigue detectando por su binario o por su estado (`config.json`, `session-state/`, etc.), y una carpeta que solo tenga skills no basta para darlo por instalado.
- **`TestSharesSkillsDirEndsWithCopilotHome`** sigue fijando la divergencia, ahora con el motivo y la fuente en su comentario.

## Alcance autorizado

Comentarios de `internal/agents/vscode/`, `internal/agents/copilotcli/`, `internal/agents/skills_dir.go` e `internal/components/uninstall/workspace.go`. Comentarios de `internal/agents/skills_dir_shared_test.go` y de `internal/components/uninstall/copilot_cli_test.go`. Y `docs/agents.md` y `docs/platforms.md`. No cambia el comportamiento.

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y el usuario no ha elegido otro modo. Comprobaciones: `go test` de los paquetes tocados. La suite entera la pasa la CI.

## Entrega

`single-pr`: una PR con un commit por tarea. Lleva la etiqueta `type:docs` porque no cambia el comportamiento. El seguimiento previsto era `type:feature` si se migraba.

## Tareas

- [x] T1 · Documentar que las skills y los agentes globales de VS Code Copilot siguen en `~/.copilot` aunque `COPILOT_HOME` esté definida. Cubre el comentario del adaptador de VS Code, los comentarios de la carpeta compartida (Copilot CLI, `SharesSkillsDirWithAnotherAdapter`, `skillsDirStaysInUse`), los comentarios de las dos pruebas que fijan la divergencia, `docs/agents.md` (VS Code Copilot y Copilot CLI) y `docs/platforms.md`. Ruta: inline. La lectura de la documentación oficial se delegó en un explorador web de solo lectura y después se contrastaron en el navegador los pasajes literales.

  Evidencia:
  - `gofmt -l` sin salida.
  - `go vet` de los cuatro paquetes: ok.
  - `go test ./internal/agents/ ./internal/agents/vscode/ ./internal/agents/copilotcli/ ./internal/components/uninstall/`: ok.

## Criterio de terminado

- La decisión y sus fuentes quedan en este documento, en el comentario del adaptador y en `docs/agents.md`.
- `go test ./internal/agents/ ./internal/agents/vscode/ ./internal/agents/copilotcli/ ./internal/components/uninstall/` en verde.

## Progreso

- Worktree creado desde `origin/main` (`0901a6e1`).
- T1 `5fa77d05`, hecha y comprobada (ver la evidencia de la tarea).

## No verificado

- No hay ninguna prueba contra un VS Code real con `COPILOT_HOME` definida. Si el arnés de VS Code sí respetase `COPILOT_HOME` para las skills, quien use solo VS Code con esa variable no las vería en `~/.copilot/skills`. Con Copilot CLI instalado también, las skills están en las dos carpetas.
- Para comprobarlo a mano: define `COPILOT_HOME`, deja una skill de prueba en `$COPILOT_HOME/skills/<nombre>/SKILL.md` y abre **Chat: Open Customizations** en una sesión del arnés Copilot. Si la skill aparece, hay que reabrir la decisión.

## Siguiente paso

Abrir la PR (`type:docs`), esperar la CI en verde, fusionar con merge commit y ejecutar `done`.
