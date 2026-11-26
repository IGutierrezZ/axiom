# Especificación de Requerimientos: Desacoplamiento Visual y Limpieza de Marca (Axiom)

## Propósito

Definir de forma exhaustiva y verificable los requerimientos y escenarios de prueba para la estandarización de nombres en adaptadores. Las capacidades de supresión del logo de OpenCode y de no-intrusión en temas se retiraron en C-08-f junto con el logo y los temas.

---

## 3. Capacidad: `adapter-brand-alignment`

Alineación de nombres de archivos de configuración e instrucciones hacia el estándar de Axiom.

### Requirement: Rutas de instrucciones con nomenclatura Axiom (RF-VIS-01)
Los adaptadores de agentes DEBEN utilizar rutas y nombres de archivos que contengan el prefijo o identificador `axiom` en lugar de `gentle-ai`.
(Previously: también fijaba la ruta del steering de Kiro IDE, `~/.kiro/steering/axiom.md`, agente retirado por C-08-d.)

#### Scenario: Archivo de instrucciones en VS Code Copilot
- **DADO** el adaptador de VS Code Copilot
- **CUANDO** se consulta la ruta del archivo de system prompt mediante `SystemPromptFile()`
- **ENTONCES** el nombre del archivo retornado es `axiom.instructions.md` (ubicado en la carpeta de prompts del usuario)

#### Scenario: Directorio de modelos en Pi
- **DADO** el adaptador de Pi
- **CUANDO** se resuelve la configuración de modelos
- **ENTONCES** la ruta de configuración apunta a `~/.pi/axiom/`

