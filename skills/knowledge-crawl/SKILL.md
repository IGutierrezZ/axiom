---
name: knowledge-crawl
description: "Trigger: knowledge crawl, crawling de codigo, analizar unidad de trabajo, catalogacion profunda, descomponer modulo. Analyze a codebase unit from crawl-job.json using Serena AST and symbols, generating canonical OpenSpec specifications for record-unit."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

## Activation Contract

Carga esta skill cuando se te asigne una unidad de trabajo proveniente de un plan determinista de crawling (`axiom knowledge crawl --plan`), o cuando debas analizar en profundidad un módulo o subsistema de código para generar o enriquecer su especificación viva canónica.

---

## Hard Rules

1. **Trabajar sobre la Unidad Asignada:**
   - La unidad de trabajo asignada contiene la lista acotada de archivos, firmas públicas e interfaces del módulo en `.axiom/knowledge/crawl-job.json`.
   - Limita tu análisis a los componentes de esa unidad para garantizar aislamiento y evitar solapamientos.

2. **Extracción Semántica con Serena:**
   - Utiliza Serena (`get_ast`, `find_symbol`, `search_definitions`) para extraer con fidelidad absoluta:
     - Nombres de structs, interfaces, enums y tipos de datos.
     - Métodos públicos y contratos de interfaces.
     - Constantes y errores tipados que definen invariantes de negocio.

3. **Estructura Canónica OpenSpec Exigida:**
   El resultado semántico de la unidad debe entregarse en formato Markdown canónico con la siguiente estructura:
   ```markdown
   # Especificación Viva: <Nombre del Módulo>

   > **Dominio:** `<slug-del-dominio>`  
   > **Estado:** Validado por análisis semántico profundo  

   ---

   ## Propósito y Responsabilidades del Dominio
   <Descripción conceptual clara de la responsabilidad única del módulo.>

   ## Modelos y Contratos Clave
   - `StructName`: <descripción e invariantes>
   - `InterfaceName`: <contrato y métodos>

   ## Requerimientos y Reglas de Negocio

   ### Requirement: <Nombre de la Regla Principal> (REQ-01)
   <Descripción de la regla y comportamiento observado en código.>

   #### Scenario: <Caso de Uso Canónico>
   - **DADO** <precondición>
   - **CUANDO** <acción del sistema>
   - **ENTONCES** <resultado verificable>
   ```

4. **Registro Atómico a través del Arnés Axiom:**
   - Una vez generado el análisis de la unidad, guárdalo en un fichero temporal y regístralo a través del comando canónico de Axiom:
     ```bash
     axiom knowledge crawl --record-unit --unit "<unit-id>" --input "<ruta-al-analisis.md>"
     ```
   - Si la unidad no puede analizarse por código corrupto o dependencias ausentes, regístrala como fallida:
     ```bash
     axiom knowledge crawl --record-failure --unit "<unit-id>" --input "<motivo-del-fallo>"
     ```
   - Nunca escribas directamente en `.axiom/knowledge/crawl-job.json` a mano; el arnés garantiza la actualización atómica del estado.

---

## Output Contract

Al completar una unidad, reporta al invocador:
- ID de la unidad procesada.
- Símbolos e interfaces analizados.
- Reglas de negocio identificadas (`REQ-XX`).
- Estado de registro devuelto por `axiom knowledge crawl --record-unit`.
