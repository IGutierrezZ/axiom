<!-- Especificación Viva generada a partir de '2026-09-16-inc-12-unified-axiom-user-state-and-env' -->

# Especificación de Requerimientos: Unificación de Estado en ~/.axiom y Variables AXIOM_* (INC-12)

## Propósito

Definir de forma verificable los requerimientos y escenarios para el almacenamiento centralizado del estado del usuario en `~/.axiom/`, la migración defensiva desde `~/.gentle-ai/` y la precedencia de variables de entorno con prefijo `AXIOM_*`.

---

## 1. Capacidad: `axiom-home-consolidation`

Centralización de los directorios de datos de usuario bajo `~/.axiom/`.

### Requirement: Ubicación autoritativa de estado y launchers en ~/.axiom (REQ-12.1)
El sistema DEBE utilizar la carpeta `~/.axiom/` para almacenar `state.json`, los binarios generados en `bin/`, las copias de seguridad en `backups/` y la caché en `cache/`.

#### Scenario: Creación de archivo de estado en ~/.axiom
- **DADO** una instalación de componentes en un equipo nuevo
- **CUANDO** se persiste la selección de agentes y componentes
- **ENTONCES** el archivo escrito es `~/.axiom/state.json`
- **Y** no se crea ninguna carpeta `~/.gentle-ai/`

#### Scenario: Ubicación de launchers en ~/.axiom/bin
- **DADO** la activación de subagentes en segundo plano de OpenCode
- **CUANDO** se generan los scripts de lanzamiento
- **ENTONCES** se escriben bajo `~/.axiom/bin/` (`opencode`, `opencode.cmd`, `opencode.ps1`)

---

## 2. Capacidad: `legacy-state-migration`

Migración transparente y sin pérdida de datos para entornos preexistentes.

### Requirement: Migración defensiva desde ~/.gentle-ai (REQ-12.2)
Si al resolver el estado no existe `~/.axiom/state.json` pero existe `~/.gentle-ai/state.json`, el sistema DEBE migrar automáticamente los contenidos hacia `~/.axiom/` antes de procesar la solicitud.

#### Scenario: Migración automática en el primer arranque
- **DADO** un equipo con `~/.gentle-ai/state.json` configurado y sin carpeta `~/.axiom/state.json`
- **CUANDO** se invoca cualquier comando de Axiom que lee el estado
- **ENTONCES** el archivo es copiado a `~/.axiom/state.json`
- **Y** la ejecución continúa normalmente utilizando el nuevo archivo migrado

---

## 3. Capacidad: `axiom-env-priority`

Prioridad y compatibilidad hacia atrás en variables de entorno del sistema.

### Requirement: Precedencia de variables AXIOM_* sobre GENTLE_AI_* (REQ-12.3)
Para cada variable de configuración soportada, el sistema DEBE comprobar primero el valor con prefijo `AXIOM_*`. Si no está definido, evaluará el valor con prefijo `GENTLE_AI_*` como respaldo.

#### Scenario: Variable AXIOM_* toma precedencia
- **DADO** `AXIOM_NO_ANIMATION=1` y `GENTLE_AI_NO_ANIMATION=0` definidos en el entorno
- **CUANDO** se consulta la política de animación de la interfaz
- **ENTONCES** el sistema adopta el valor `1` (animación desactivada) proveniente de `AXIOM_NO_ANIMATION`

#### Scenario: Fallback a GENTLE_AI_* si AXIOM_* no está definido
- **DADO** únicamente `GENTLE_AI_OPENCODE_BACKGROUND_SUBAGENTS=true` definido en el entorno
- **CUANDO** se resuelve la intención de subagentes en segundo plano
- **ENTONCES** el sistema adopta el valor `true`
### Requirement: La Lectura del Estado Tolera Restos de RDD (REQ-46.5)

La carga de `state.json` (estado global y estado del hub) DEBE ignorar sin error el campo `rdd_mode` (y `rdd_mode_recorded_at`) y cualquier identificador de skill que ya no exista en el catálogo, entre ellos `rdd-defect-workflow`, tanto en la selección persistida como en el modelo de selección de `sync`. El sistema NO DEBE emitir aviso ni error por ello, NO DEBE instalar la skill desconocida y NO DEBE volver a escribir esos campos: en la siguiente escritura del estado desaparecen. El resto de campos del estado se conserva sin cambios.

#### Scenario: Un `state.json` con `rdd_mode` carga sin error

- **DADO** un `state.json` con `"rdd_mode": "on"`, `"rdd_mode_recorded_at": "2026-01-01T00:00:00Z"` y otros campos válidos
- **CUANDO** se carga el estado
- **ENTONCES** la carga termina sin error
- **Y** los demás campos conservan sus valores

#### Scenario: Una selección con `rdd-defect-workflow` no rompe `sync`

- **DADO** un `state.json` cuya selección de skills incluye `rdd-defect-workflow`
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** `sync` termina sin error
- **Y** no se instala ninguna skill `rdd-defect-workflow`
- **Y** el resto de skills seleccionadas se instalan como antes

#### Scenario: La siguiente escritura elimina los restos

- **DADO** un `state.json` con `rdd_mode` leído sin error
- **CUANDO** el sistema vuelve a persistir el estado
- **ENTONCES** el fichero resultante no contiene `rdd_mode` ni `rdd_mode_recorded_at`
- **Y** el resto de campos se conserva

### Requirement: Aislamiento del Directorio de Trabajo (CWD) en la Gestión de Artefactos (REQ-46.8)

La resolución de rutas de perfiles de agentes y persona DEBE realizarse contra sus directorios raíz canónicos (`homeDir` o `workspaceRoot`) sin depender del CWD.
(Trasladado desde REQ-18.4 de `rdd-decoupling-v3-stability`, retirada con RDD (C-01-e).)

#### Scenario: Sincronización de agentes desde un subdirectorio anidado

- **DADO** un workspace de Axiom e invocación desde un subdirectorio
- **CUANDO** se ejecuta una sincronización (`axiom sync`)
- **ENTONCES** los archivos se escriben en sus ubicaciones canónicas absolutas sin crear artefactos relativos al CWD
