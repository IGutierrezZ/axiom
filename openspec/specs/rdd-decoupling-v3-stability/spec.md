# Especificación Viva: Desacoplamiento de RDD y Parches de Estabilidad Upstream v3

> **Dominio:** `rdd-decoupling-v3-stability`  
> **Versión Canónica:** 1.0.0  
> **Estado:** Vigente  
> **Idioma:** Español (Castellano peninsular)  

---

## 1. Contexto y Propósito

Esta especificación consolida la arquitectura canónica de Axiom respecto a la independencia del ciclo de vida Spec-Driven Development (SDD) frente a Review-Driven Development (RDD), así como la absorción de correcciones críticas de estabilidad provenientes de Gentle-AI v3:

1. **Desacoplamiento Estructural de RDD:** La proyección de estado SDD (`sdd status`) no contiene ofertas ni dependencias forzadas con transacciones de revisión. El avance hacia el archivado (`archive: ready`) está desacoplado de la participación de RDD y depende únicamente de la compleción de tareas y el veredicto favorable en `verify-report.md`.
2. **Independencia de Herramientas de Revisión:** Las utilidades de revisión de código (`axiom review ...`) son herramientas opt-in independientes gobernadas por el usuario que no interfieren en la máquina de estados SDD.
3. **Resiliencia en Windows:** Las aserciones y procesamiento de respuestas JSON de la CLI bajo Windows emplean decodificación estructurada en tipos Go nativos para evitar fallos por barras invertidas escapadas.
4. **Aislamiento de CWD en Gestión de Artefactos:** Los artefactos de agentes y persona se resuelven contra directorios canónicos absolutos (`homeDir`, `componentInjectionDir`) sin depender del directorio de trabajo actual.
5. **Saneamiento de Presets:** Los presets de usuario general (`selectableFoundationSkills`) contienen exclusivamente skills de producto, excluyendo las skills de flujos internos de colaboración (`contributorSkills`).
6. **Resiliencia de Engram:** Protocolo unificado de recuperación determinista ante el estado `ambiguous_project`.

---

## 2. Requerimientos Canónicos y Escenarios BDD

### Requirement: Desacoplamiento de RDD en la Proyección de Estado SDD (REQ-18.1)
El motor de estados en `internal/sddstatus` DEBE omitir de forma limpia el bloque `reviewOffer` y llamadas acopladas a transacciones de revisión. La preparación para el archivado (`archive: ready`) DEBE evaluarse de forma independiente a la presencia de revisiones de código.

#### Scenario: Proyección de estado SDD tras verificación exitosa sin reviewOffer forzado
- **DADO** un incremento en fase de verificación con tareas completadas al 100% y `verify-report.md` con veredicto PASS
- **CUANDO** se ejecuta la proyección de estado `axiom sdd status [cambio] --json`
- **ENTONCES** el documento JSON omitirá la clave `reviewOffer`
- **Y** el campo `dependencies.archive` se evaluará como `ready`

---

### Requirement: Independencia Operativa de las Herramientas RDD (REQ-18.2)
RDD está suspendido (C-01-c) hasta su retirada (C-01-e). Las herramientas de revisión (`axiom review ...`) NO DEBEN condicionar la máquina de estados SDD. El modo efectivo de RDD DEBE ser siempre `off`: `axiom review mode enable` DEBE fallar en todos los alcances con un mensaje que indique que RDD no está disponible, y `axiom review mode status` DEBE informar `off` aunque haya preferencias `on` guardadas. Axiom NO DEBE proyectar prosa de RDD en orquestadores ni en la guía de agentes, NO DEBE instalar los agentes `review-*` y NO DEBE registrar el hook `axiom review stop-hook`; `axiom review stop-hook` DEBE terminar sin error y sin efecto para los proyectos que aún lo tengan registrado. Solo un binario compilado con la etiqueta `rdd_harness` (bench, e2e y CI) PUEDE activar RDD.
(Previously: las herramientas `axiom review ...` eran comandos opt-in gobernados por el usuario)

#### Scenario: Activar RDD falla en todos los alcances
- **DADO** el binario `axiom` sin la etiqueta `rdd_harness`
- **CUANDO** el usuario ejecuta `axiom review mode enable --scope global`, `--scope project` o `--scope clone`
- **ENTONCES** la orden termina con error indicando que RDD no está disponible
- **Y** no guarda ninguna preferencia

#### Scenario: Las preferencias guardadas no activan RDD
- **DADO** una preferencia `on` guardada en el ámbito global, en el proyecto del hub o en el clon
- **CUANDO** se ejecuta `axiom review mode status`
- **ENTONCES** el modo efectivo es `off`

#### Scenario: La sincronización no instala RDD
- **DADO** un proyecto limpio
- **CUANDO** se ejecuta `axiom sync --scope workspace`
- **ENTONCES** no se escriben agentes `review-*`, no se registra el hook de parada de revisión y los ficheros gestionados no contienen prosa de RDD

#### Scenario: El hook de parada antiguo no falla
- **DADO** un proyecto que aún registra `axiom review stop-hook`
- **CUANDO** el runtime lo invoca
- **ENTONCES** termina con código 0 sin efecto

#### Scenario: Las herramientas de revisión no alteran las fases SDD
- **DADO** el binario `axiom`
- **CUANDO** el usuario ejecuta comandos bajo `axiom review`
- **ENTONCES** las fases SDD del workspace no cambian

### Requirement: Resiliencia en Decodificación de Rutas Windows en Salidas JSON de SDD (REQ-18.3)
Los consumidores y aserciones de la salida JSON de `sdd status` DEBEN decodificar estructuradamente en tipos Go nativos (`StatusV2Projection`) en lugar de hacer matching textual simple de rutas en Windows.

#### Scenario: Validación de rutas permitidas bajo Windows
- **DADO** un entorno de ejecución Windows con rutas que contienen separadores de barra invertida (`\`)
- **CUANDO** se valida la salida JSON de `RunSDDStatus`
- **ENTONCES** la validación se realiza mediante `json.Unmarshal`, tolerando el escape de backslashes sin fallos de aserción

---

### Requirement: Aislamiento del Directorio de Trabajo (CWD) en la Gestión de Artefactos (REQ-18.4)
La resolución de rutas de perfiles de agentes y persona DEBE realizarse contra sus directorios raíz canónicos (`homeDir` o `workspaceRoot`) sin depender del CWD.

#### Scenario: Sincronización de agentes desde un subdirectorio anidado
- **DADO** un workspace de Axiom e invocación desde un subdirectorio
- **CUANDO** se ejecuta una sincronización (`axiom sync`)
- **ENTONCES** los archivos se escriben en sus ubicaciones canónicas absolutas sin crear artefactos relativos al CWD

---

### Requirement: Saneamiento del Preset Predeterminado de Skills (REQ-18.5)
El catálogo de skills distribuido en el preset básico de Axiom DEBE contener únicamente skills de producto para usuarios, segregando las skills de colaboración interna del repositorio (`contributorSkills`).

#### Scenario: Inspección del preset básico de skills
- **DADO** el catálogo de skills en `internal/components/skills`
- **CUANDO** se consulta el listado del preset por defecto
- **ENTONCES** contiene únicamente skills fundacionales de producto y excluye las de contribución interna

---

### Requirement: Resiliencia de Protocolo Engram ante Ambiguous Project (REQ-18.6)
El protocolo y directrices de integración con Engram MCP DEBEN documentar y contemplar el tratamiento ordenado del estado `ambiguous_project` en el apretón de manos inicial.

#### Scenario: Manejo de ambigüedad de proyecto en handshake de sesión Engram
- **DADO** un entorno multi-proyecto con Engram MCP
- **CUANDO** el agente detecta una indicación de proyecto ambiguo
- **ENTONCES** el protocolo instruye la especificación unívoca del identificador de proyecto mediante `--project=<nombre>` antes de reintentar
