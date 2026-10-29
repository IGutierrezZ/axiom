# Especificación de Requerimientos: Barrido Autónomo Knowledge y Siembra Inicial (INC-23)

> **Incremento:** `inc-23-knowledge-profile` · **Capacidad:** `knowledge-sweep-command` · **Tipo:** capacidad nueva
> **Fase del Roadmap:** Fase 4 — Gobernanza de Agentes, Integración de Conocimiento y Descubrimiento Semántico  
> **Responsabilidad:** Hub & Init / Dominio Knowledge / CLI Commands / OpenSpec Living Documentation / MCP Integration  
> **Idioma:** Español (castellano peninsular)  

## Propósito

Permite realizar un barrido inicial rápido sobre los repositorios del proyecto para inferir la arquitectura técnica y los módulos funcionales sin bloqueos interactivos.

### Requirement: Invocación Autónoma Headless (REQ-23.7)

El subcomando `axiom knowledge sweep` DEBE soportar ejecución autónoma no interactiva (`--headless`, `--json`), adecuada para ser invocada desde el backend hacia CLIs de agentes de codificación (Claude Code, Codex, Copilot, Gemini CLI) sin requerir confirmación por terminal.

#### Scenario: Ejecución headless de barrido emitiendo JSON
- **DADO** un workspace con código preexistente
- **CUANDO** el backend ejecuta `axiom knowledge sweep --json`
- **ENTONCES** el comando concluye con código de salida 0 sin solicitar confirmación interactiva
- **Y** emite un JSON estructurado con la arquitectura técnica y los módulos funcionales detectados

---

### Requirement: Detección Técnica Multidimensional (REQ-23.8)

El barrido DEBE inspeccionar el árbol de código y las definiciones AST/símbolos disponibles (vía Serena o detector nativo) para identificar:
- Lenguajes de programación principales y secundarios.
- Frameworks y dependencias clave.
- Puntos de entrada (`main`, controladores, handlers, configuraciones).
- Estructura y capas de la arquitectura (monolito, microservicios, hexagonal, MVC, etc.).

#### Scenario: Identificación de stack y puntos de entrada
- **DADO** un repositorio con un proyecto Go y frontend React
- **CUANDO** se ejecuta el barrido `axiom knowledge sweep`
- **ENTONCES** el reporte técnico incluye `go` y `typescript/react` en lenguajes/frameworks
- **Y** señala los puntos de entrada principales detectados en el sistema

---

### Requirement: Inferencia Funcional de Módulos y Dominios (REQ-23.9)

El barrido DEBE agrupar la funcionalidad del código en módulos o dominios funcionales preliminares, deduciéndolos a partir de nombres de directorios, servicios, entidades de dominio y rutas/endpoints públicos.

#### Scenario: Inferencia de módulos de negocio
- **DADO** una estructura de código con directorios `billing/`, `auth/` y `catalog/`
- **CUANDO** finaliza el barrido
- **ENTONCES** el resultado clasifica preliminarmente los dominios `billing`, `auth` y `catalog` con sus componentes principales

---

### Requirement: Política No Bloqueante y Sobre Diferido de Ambigüedades (REQ-23.10)

Durante el barrido, el sistema NO DEBE detenerse ni formular preguntas al usuario ante dudas, incoherencias técnicas o código huérfano. Todas las lagunas, incertidumbres o ambigüedades detectadas DEBEN recopilarse en un sobre estructurado (`ambiguities`) que se devuelve al concluir el proceso.

#### Scenario: Detección de ambigüedades sin bloqueo
- **DADO** un repositorio con archivos huérfanos o dos frameworks en conflicto aparente
- **CUANDO** se ejecuta el barrido
- **ENTONCES** el proceso completa la ejecución sin pausar
- **Y** la salida incluye un apartado de `ambigüedades` describiendo las dudas encontradas para aclaración posterior

---

### Requirement: Siembra Inicial en OpenSpec y Persistencia en Engram (REQ-23.11)

Al finalizar con éxito el barrido, el comando DEBE:
1. Sembrar borradores de especificación viva en `openspec/specs/<dominio>/spec.md` para cada módulo funcional detectado.
2. Actualizar el índice general `openspec/INDEX.md`.
3. Persistir un resumen estructurado del análisis en Engram bajo el tópico `knowledge/sweep` con tipo `architecture`.

#### Scenario: Persistencia y catalogación del barrido inicial
- **DADO** un barrido completado con éxito
- **CUANDO** concluye la fase de persistencia
- **ENTONCES** se generan los archivos `spec.md` para los dominios detectados en `openspec/specs/`
- **Y** `openspec/INDEX.md` refleja los dominios agregados
- **Y** se guarda una memoria persistente en Engram con el mapa del proyecto
