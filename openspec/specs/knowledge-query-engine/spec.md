# Especificación de Requerimientos: Consulta Spec-First con Auto-Enriquecimiento de la Spec Viva (INC-23)

> **Incremento:** `inc-23-knowledge-profile` · **Capacidad:** `knowledge-query-engine` · **Tipo:** capacidad nueva
> **Fase del Roadmap:** Fase 4 — Gobernanza de Agentes, Integración de Conocimiento y Descubrimiento Semántico  
> **Responsabilidad:** Hub & Init / Dominio Knowledge / CLI Commands / OpenSpec Living Documentation / MCP Integration  
> **Idioma:** Español (castellano peninsular)  

## Propósito

Permite formular consultas técnicas o funcionales sobre el proyecto, respondiendo con fundamentación y enriqueciendo automáticamente las especificaciones vivas.

### Requirement: Subcomando de Consulta Técnica/Funcional (REQ-23.12)

El sistema DEBE proveer el comando `axiom knowledge query "<pregunta>" [--type=technical|functional]` para recibir consultas en lenguaje natural orientadas a la arquitectura o a las reglas de negocio del proyecto.

#### Scenario: Invocación de consulta funcional
- **DADO** un workspace con documentación y código
- **CUANDO** se ejecuta `axiom knowledge query "¿Cómo se procesan los reembolsos?" --type=functional`
- **ENTONCES** el motor procesa la consulta bajo la lente funcional

---

### Requirement: Resolución Spec-First con Cortocircuito Inmediato (REQ-23.13)

El motor de consulta DEBE evaluar primero si la pregunta se encuentra resuelta con suficiente detalle y precisión dentro de las especificaciones vivas existentes (`openspec/specs/`). Si la respuesta está contenida en la spec, el motor DEBE responder directamente a partir de ella sin necesidad de inspeccionar el código ni invocar conectores semánticos.

#### Scenario: Respuesta directa desde la spec viva sin tocar código
- **DADO** una especificación viva `openspec/specs/billing/spec.md` que detalla las reglas de reembolso
- **CUANDO** se consulta `axiom knowledge query "¿Cuáles son las condiciones de reembolso?"`
- **ENTONCES** el motor responde utilizando la especificación viva
- **Y** no realiza llamadas a Serena ni a CodeGraph para inspeccionar código

---

### Requirement: Triangulación Profunda con Evidencias ante Lagunas en la Spec (REQ-23.14)

Si la consulta no está cubierta en la spec viva o requiere detalle de implementación no documentado, el motor DEBE realizar una investigación profunda triangulando:
1. Símbolos y AST mediante Serena.
2. Jerarquía de llamadas y dependencias mediante CodeGraph.
3. Decisiones y descubrimientos previos en la memoria Engram.
4. Código fuente correspondiente en los repositorios locales.

#### Scenario: Consulta que requiere inspección profunda de código
- **DADO** una consulta cuya respuesta no figura en `openspec/specs/`
- **CUANDO** el motor detecta la ausencia en la spec
- **ENTONCES** consulta a Serena para localizar las firmas relevantes y a CodeGraph para rastrear los invocadores
- **Y** examina las líneas de código afectadas

---

### Requirement: Contrato de Respuesta en Dos Niveles (REQ-23.15)

Toda respuesta emitida por `axiom knowledge query` DEBE estructurarse estrictamente en dos bloques:
1. **Respuesta Directa:** Explicación clara, concisa y rigurosa en castellano que resuelve la duda funcional o técnica.
2. **Evidencias Concretas:** Lista detallada de evidencias que sustentan la respuesta (ficheros, rangos de línea, declaraciones de tipos y relaciones de callers/callees identificadas).

#### Scenario: Formato de la respuesta estructurada
- **DADO** una consulta resuelta mediante evidencias de código
- **CUANDO** se presenta la salida al usuario
- **ENTONCES** la primera sección contiene la respuesta directa a la pregunta formulada
- **Y** la segunda sección lista las evidencias con rutas a los ficheros, números de línea y símbolos analizados

---

### Requirement: Auto-Enriquecimiento Obligatorio de la Spec Viva (REQ-23.16)

Cuando una consulta aporte nuevo conocimiento, reglas de negocio o detalles arquitectónicos que no figuraban previamente en la especificación viva, el motor DEBE:
1. Sintetizar el nuevo requerimiento o aclaración en el archivo `openspec/specs/<dominio>/spec.md` correspondiente.
2. Actualizar el índice general `openspec/INDEX.md`.
3. Registrar la observación aprendida en Engram con tipo `discovery` o `architecture`.
Si la consulta no generó conocimiento nuevo (fue resuelta por la spec existente), no se reescriben los archivos.

#### Scenario: Auto-enriquecimiento de la spec tras investigación
- **DADO** una consulta investigada en el código cuya regla no existía en `openspec/specs/`
- **CUANDO** concluye la respuesta
- **ENTONCES** el motor incorpora el nuevo requerimiento a `openspec/specs/<dominio>/spec.md`
- **Y** actualiza `openspec/INDEX.md`
- **Y** asienta el descubrimiento en la memoria Engram con `mem_save`
