# Delta for axiom-user-state-and-env

## ADDED Requirements

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
