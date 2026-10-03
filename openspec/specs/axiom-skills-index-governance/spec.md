# Especificación Viva: Gobernanza del Índice Unificado de Skills

> **Dominio:** `axiom-skills-index-governance`  
> **Versión Canónica:** 1.1.0  
> **Estado:** Vigente  
> **Idioma:** Español (Castellano peninsular)

---

## 1. Capacidad: `axiom-skills-index-governance`

Gobernanza del catálogo de habilidades (skills) de agentes en tres destinos sincronizados desde un único escaneo (`.atl/skill-registry.md`, sección gestionada `## Skills` en `AGENTS.md` y tópico persistente en Engram MCP), con una vista completa para el registro local y una vista versionada para los destinos compartidos, soporte de adopción atómica mediante marcadores canónicos y gancho no transaccional tras promociones en `autoskill`.

### Requirement: Superficie CLI del índice de skills (REQ-22.10)

El sistema DEBE exponer el subcomando `axiom skill index <refresh|list>` dentro del grupo `axiom skill`, diferenciando claramente en ayuda y ejecución entre `skill list` (skills activas y buzón de autoskill) y `skill index list` (inventario unificado indexado).

#### Scenario: Consulta del índice desde CLI
- **DADO** un workspace con skills instaladas y globales
- **CUANDO** el usuario ejecuta `axiom skill index list`
- **ENTONCES** se muestra la lista formateada con nombre, ámbito (`project`/`user`) y ruta de cada skill

---

### Requirement: Regeneración unificada en tres destinos desde un único escaneo (REQ-22.11)

La ejecución de `axiom skill index refresh` DEBE escanear las skills una sola vez, construir a partir de ese escaneo dos vistas y actualizar atómicamente:
1. El registro local en disco `.atl/skill-registry.md` (ignorado por git), con la **vista completa**: todas las skills escaneadas, incluidas las de ámbito usuario (`$HOME`, con rutas absolutas) y las copias locales que generan los agentes.
2. La sección delimitada `## Skills` en el archivo `AGENTS.md` de la raíz del workspace, con la **vista versionada** y rutas relativas al workspace.
3. El tópico de persistencia `skill-registry` en Engram MCP mediante cliente stdio acotado, con la misma **vista versionada**.

`AGENTS.md` se versiona y lo consumen todos los colaboradores, por lo que su contenido DEBE ser el mismo en cualquier clon. Una skill pertenece a la vista versionada si, y solo si, se cumplen a la vez estas dos condiciones:
- Su `SKILL.md` está dentro del workspace. Las skills de ámbito usuario y las raíces absolutas o con `../` fuera del workspace solo figuran en la vista completa.
- Git no la ignora, según `git check-ignore` sin `--no-index`: un fichero rastreado por git dentro de un directorio ignorado cuenta como versionado.

Las entradas no versionables DEBEN descartarse antes de deduplicar por nombre, de modo que, si una skill existe como copia versionada y como copia ignorada, figure la versionada. La deduplicación se aplica por separado a cada vista.

La clasificación DEBE hacerse con una única invocación de git por regeneración. Cuando git no esté disponible, el directorio no sea un repositorio o git no dé respuesta (error, binario ausente o tiempo agotado), NO DEBE aplicarse filtro de git y toda skill situada dentro del workspace DEBE contar como versionable. Un fallo de git NO DEBE ser nunca fatal ni impedir la regeneración.

La huella de caché DEBE incorporar qué entradas son versionables, de modo que un cambio de versionabilidad (por ejemplo, editar `.gitignore`) provoque la regeneración sin necesidad de `--force`.

Un fallo de conexión con Engram MCP NO DEBE impedir la actualización de los destinos locales en disco ni producir un código de salida fallido.

#### Scenario: Regeneración exitosa en los tres destinos
- **DADO** el workspace con agentes y servidor Engram activo
- **CUANDO** se invoca `axiom skill index refresh`
- **ENTONCES** los tres destinos reflejan el conjunto actual de skills que les corresponde según su vista
- **Y** el comando retorna código de salida `0`

#### Scenario: Copia local ignorada excluida de AGENTS.md y conservada en el registro local
- **DADO** un repositorio git que ignora `.claude/` y una skill `local-only` presente únicamente en `.claude/skills/local-only/`
- **CUANDO** se ejecuta `axiom skill index refresh`
- **ENTONCES** `.atl/skill-registry.md` lista `local-only`
- **Y** ni la sección `## Skills` de `AGENTS.md` ni el tópico de Engram la listan

#### Scenario: La copia versionada prevalece sobre la copia ignorada
- **DADO** la skill `go-testing` en un directorio versionado (`internal/assets/skills/go-testing/`) y una copia con otra descripción en `.claude/skills/go-testing/`, ignorada por git
- **CUANDO** se ejecuta `axiom skill index refresh`
- **ENTONCES** `AGENTS.md` y el tópico de Engram listan `go-testing` con la ruta relativa `internal/assets/skills/go-testing/SKILL.md` y la descripción de la copia versionada
- **Y** la sección `## Skills` queda byte a byte idéntica con o sin la copia ignorada presente

#### Scenario: Skill de ámbito usuario solo en el registro local
- **DADO** una skill instalada en el directorio de usuario (`$HOME/.claude/skills/personal/`)
- **CUANDO** se ejecuta `axiom skill index refresh`
- **ENTONCES** `.atl/skill-registry.md` la lista con ámbito `user` y ruta absoluta
- **Y** `AGENTS.md` y el tópico de Engram no la listan, porque su `SKILL.md` está fuera del workspace

#### Scenario: Fichero ignorado pero rastreado cuenta como versionado
- **DADO** una skill bajo un directorio que `.gitignore` ignora, cuyo `SKILL.md` está rastreado por git
- **CUANDO** se ejecuta `axiom skill index refresh`
- **ENTONCES** `AGENTS.md` y el tópico de Engram la listan con su ruta relativa al workspace

#### Scenario: Directorio sin git o sin respuesta de git
- **DADO** un workspace que no es un repositorio git, o en el que git no está disponible o agota el tiempo de espera
- **CUANDO** se ejecuta `axiom skill index refresh`
- **ENTONCES** toda skill situada dentro del workspace figura en `AGENTS.md` y en el tópico de Engram con su ruta relativa, y las situadas fuera del workspace siguen sin figurar
- **Y** el comando retorna código de salida `0`

#### Scenario: Editar .gitignore invalida la caché
- **DADO** un refresco previo en el que una skill de `.claude/skills/` figuraba en `AGENTS.md` porque git no la ignoraba
- **CUANDO** se añade `.claude/` a `.gitignore` y se ejecuta `axiom skill index refresh` sin `--force`
- **ENTONCES** la huella de caché cambia y los destinos se regeneran
- **Y** `AGENTS.md` y el tópico de Engram dejan de listar la skill, mientras que `.atl/skill-registry.md` la conserva

---

### Requirement: Reemplazo atómico y marcado de ## Skills en AGENTS.md (REQ-22.12)

La sección `## Skills` de `AGENTS.md` DEBE delimitarse por los marcadores canónicos:
```markdown
<!-- axiom:skills-index -->
## Skills

| Skill | Trigger / description | Scope | Path |
| --- | --- | --- | --- |
...
<!-- /axiom:skills-index -->
```
El motor `filemerge.InjectMarkdownSection` DEBE operar de forma atómica sobre el bloque, preservando intacto el resto de secciones del documento y elevando marcadores legados si estuvieran presentes.

#### Scenario: Idempotencia en la sustitución de marcadores
- **DADO** un `AGENTS.md` con marcadores canónicos
- **CUANDO** se regenera el índice consecutivamente sin cambios de skills
- **ENTONCES** el archivo permanece byte a byte idéntico
- **Y** ninguna sección circundante se altera

---

### Requirement: Disparo automático del índice desde autoskill (REQ-22.13)

Cuando el gestor de autoskills apruebe una propuesta (`Manager.Approve`), el sistema DEBE disparar automáticamente la regeneración del índice unificado para que la nueva skill esté disponible de inmediato en todos los destinos. El gancho DEBE ser no transaccional: un eventual fallo de indexación DEBE emitirse como aviso y no debe revertir la promoción completada. Descartar una propuesta (`Reject`) NO DEBE disparar la regeneración.

#### Scenario: Aprobación de skill promueve e indexa
- **DADO** una skill candidata en el buzón transitorio
- **CUANDO** se aprueba con `axiom skill approve <nombre>`
- **ENTONCES** la skill se mueve a `skills/<nombre>/`
- **Y** el índice se actualiza automáticamente

---

### Requirement: Compatibilidad del verbo skill-registry (REQ-22.14)

El verbo `axiom skill-registry <refresh|list>` DEBE conservarse plenamente operativo y con compatibilidad byte a byte en flags, salida primaria y formato JSON, delegando en el motor unificado de `skill index`.

#### Scenario: Invocación por scripts externos
- **DADO** un plugin que invoca `axiom skill-registry refresh --quiet --no-gitignore`
- **CUANDO** se ejecuta la orden
- **ENTONCES** el comando se procesa con éxito con código de salida `0`

---

### Requirement: Raíces de skills declaradas en axiom.yaml (REQ-22.15)

El archivo `axiom.yaml` PUEDE declarar `workspace.skill_roots`: una lista de directorios versionados, relativos a la raíz del workspace, que contienen skills canónicas fuera de `skills/` (por ejemplo, `internal/assets/skills`). El escaneo de REQ-22.11 DEBE recorrerlos inmediatamente después de `skills/` y antes de los directorios propios de cada agente (`.claude/skills`, `.gemini/skills`, etc.), de modo que, ante un nombre duplicado, prevalezca la copia declarada.

Cada entrada DEBE ser una ruta relativa que permanezca dentro del workspace. Las rutas absolutas (incluidas `/x`, `\x`, `C:\x`, `C:x` y UNC), las que salen del workspace (`..`, `../x`), la ruta `.` y la cadena vacía NO DEBEN escanearse, y `axiom workspace validate` DEBE reportar cada una como error. La ausencia de la clave NO DEBE alterar el comportamiento del escaneo.

#### Scenario: La raíz declarada prevalece sobre la copia de un agente
- **DADO** un `axiom.yaml` con `workspace.skill_roots: ["internal/assets/skills"]` y la misma skill también en `.claude/skills/`
- **CUANDO** se ejecuta `axiom skill index refresh`
- **ENTONCES** la ruta indexada de la skill es la de `internal/assets/skills/`
- **Y** la copia de `.claude/skills/` no altera `AGENTS.md`

#### Scenario: Raíz declarada inválida
- **DADO** un `axiom.yaml` con `workspace.skill_roots: ["../fuera", "/abs", "."]`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta un error por cada una de las tres entradas y el espacio de trabajo no es conforme
- **Y** `axiom skill index refresh` las ignora sin fallar
