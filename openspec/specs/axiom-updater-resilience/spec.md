# Especificación Viva: Resiliencia y Gobernanza del Actualizador Autónomo de Axiom

> **Dominio:** `axiom-updater-resilience`  
> **Versión Canónica:** 1.0.0  
> **Estado:** Vigente  
> **Idioma:** Español (Castellano peninsular)

---

## 1. Capacidad: `axiom-updater-resilience`

Estrategias y salvaguardas de actualización del binario `axiom` para entornos Windows, macOS y Linux: resolución segura del destino de instalación, vía resiliente `sourceBuildUpgrade`, encadenamiento `upgrade` ➔ `sync` en Web UI y TUI, y preservación del contrato CLI solo-binario.

### Requirement: Ejecución segura de la actualización en Windows (RF-UPD-01)

El sistema DEBE aplicar, en entorno Windows, una estrategia de actualización de la herramienta del fork que no falle por discrepancia de ruta de módulo en `go.mod` y que reemplace el binario de forma segura y transparente. Cuando ninguna estrategia automatizada pueda garantizar esas dos condiciones, el sistema DEBE degradar a una instrucción de actualización manual accionable con `ManualFallbackError` y NO DEBE dejar el binario a medias ni duplicar binarios en PATH.

#### Scenario: Degradación manual segura ante destinos disjuntos
- **DADO** un entorno Windows donde el ejecutable activo en PATH difiere del destino resuelto por `go env GOBIN`/`GOPATH`
- **CUANDO** se intenta una auto-actualización
- **ENTONCES** la actualización se rehúsa antes de escribir
- **Y** se emite una instrucción manual precisa con comandos para el usuario

---

### Requirement: Predicado de identidad del módulo para Go install (RF-UPD-02)

Las instrucciones de instalación y las comprobaciones de compatibilidad DEBEN derivarse de la estructura `ToolInfo` asociada al binario (`GoModulePath`, `GoImportPath`), reconociendo la identidad del fork mediante el predicado `IsSelfToolName`/`IsSelfTool`. La función `GoInstallResolvable()` DEBE validar que la ruta declarada sea alcanzable mediante `go install` estándar antes de seleccionar esa estrategia.

#### Scenario: Ruteo condicionado por resolubilidad de módulo
- **DADO** que `GoInstallResolvable()` retorna falso para el binario `axiom`
- **CUANDO** se resuelve el método de actualización en Windows
- **ENTONCES** el actualizador selecciona la estrategia de construcción desde código fuente (`InstallSourceBuild`) o degradación manual
- **Y** no invoca `go install` contra un import path no coincidente con `go.mod`

---

### Requirement: Salvaguardas de actualización ancladas a la identidad del fork (RF-UPD-03)

Todas las salvaguardas de comprobación previa de binarios en ejecución, renombrado atómico temporal y validación de procedencia DEBEN operar sobre la identidad del fork (`axiom`) y no sobre literales heredados de upstream (`gentle-ai`).

#### Scenario: Verificación de binario en ejecución
- **DADO** una actualización en caliente de `axiom.exe`
- **CUANDO** se ejecuta el reemplazo
- **ENTONCES** se verifica la firma y el proceso bajo el nombre `axiom`
- **Y** se asegura la terminación limpia o renombramiento transitorio

---

### Requirement: Endpoint Web UI encadena upgrade ➔ sync con reporte consolidado (RF-UPD-04)

El endpoint `POST /api/ecosystem/upgrade` del Dashboard Web DEBE ejecutar secuencialmente la fase de actualización de binarios (`upgrade`) seguida de la fase de sincronización de reglas y configuraciones (`sync`), retornando un DTO consolidado con el desglose por fases (`phases.upgrade`, `phases.sync`). Si la fase `upgrade` requiere reinicio del proceso (`restart_required: true`), la fase `sync` DEBE omitirse limpiamente (`skipped_reason: "restart-required"`).

#### Scenario: Secuencia completada en el Dashboard
- **DADO** una petición `POST /api/ecosystem/upgrade` desde la Web UI
- **CUANDO** no se requiere reinicio inmediato del servidor
- **ENTONCES** se ejecuta `upgrade` y posteriormente `sync`
- **Y** la respuesta 200 OK contiene el reporte de ambas fases

---

### Requirement: No-regresión del verbo CLI axiom upgrade (RF-UPD-05)

El comando de terminal `axiom upgrade` DEBE mantenerse estrictamente como una operación **solo-binario**; NO DEBE encadenar automáticamente `sync` ni `install`. La composición de `upgrade` ➔ `sync` es exclusiva de la superficie reactiva interactiva (Web UI).
(Previously: la composición era exclusiva de las superficies reactivas interactivas (Web UI y TUI).)

#### Scenario: Ejecución de axiom upgrade desde consola
- **DADO** la invocación de `axiom upgrade` en CLI
- **CUANDO** se completa la actualización de herramientas
- **ENTONCES** solo se actualizan los binarios
- **Y** no se mutan configuraciones ni se disparan sincronizaciones de agentes

### Requirement: Identidad visible del fork en vistas de actualización (RF-UPD-06)

Todos los mensajes de confirmación, cabeceras, títulos y diálogos en la Web UI DEBEN referirse exclusivamente a la identidad de `axiom`.
(Previously: la identidad aplicaba a TUI y Web UI; el escenario comprobaba la pantalla `ScreenUpgradeSync`.)

#### Scenario: Textos de interfaz limpios
- **DADO** la vista de actualización y sincronización de la Web UI
- **CUANDO** se renderizan los diálogos de progreso
- **ENTONCES** todos los textos nombran `axiom`

### Requirement: Versión de build visible y consistente (RF-UPD-07)

El símbolo de versión en `cmd/axiom/main.go` DEBE declararse como variable mutable sin valor por defecto (`var version string`), para que el linker (`-X main.version=...`) inyecte el tag exacto en los flujos de CI y GoReleaser. La versión informada DEBE resolverse con esta prioridad:
1. el valor inyectado;
2. si no hay inyección, `debug.ReadBuildInfo().Main.Version`, que `go install …@vX` rellena, salvo que esté vacía o sea `(devel)`;
3. la constante `fallbackVersion` como último recurso.

#### Scenario: Inyección de versión en tiempo de enlace
- **DADO** una compilación con `-ldflags="-X main.version=v3.5.0"`
- **CUANDO** se ejecuta `axiom --version`
- **ENTONCES** la salida refleja `v3.5.0`

#### Scenario: Binario instalado con go install
- **DADO** un binario instalado con `go install github.com/IGutierrezZ/axiom/v3/cmd/axiom@v3.6.0`, sin ldflags
- **CUANDO** se ejecuta `axiom --version`
- **ENTONCES** la salida refleja `v3.6.0`, no la constante `fallbackVersion`

---

### Requirement: Registro durable de la versión upstream de referencia (RF-UPD-08)

El archivo de estado de usuario (`~/.axiom/state.json`) DEBE mantener la clave de auditoría `upstream_version` inicializada con el techo de referencia (`"3.4.0"`), sin ser sobreescrita ni consumida automáticamente por rutas de ejecución.

#### Scenario: Conservación de la versión upstream de referencia
- **DADO** la inicialización de estado de Axiom
- **CUANDO** se consulta `state.json`
- **ENTONCES** el registro contiene `upstream_version: "3.4.0"`
### Requirement: Reinicio pendiente detectado en el dashboard (RF-UPD-09)

El dashboard DEBE comparar la versión del proceso en ejecución con la del binario de Axiom que arrancaría al reiniciarse. Cuando difieren y el binario instalado no está por detrás de la última versión publicada, DEBE ofrecer reiniciar el dashboard y NO DEBE ofrecer ni ejecutar una actualización. Si no puede leer la versión del binario instalado, DEBE comprobar con la versión del proceso, como hasta ahora.

#### Scenario: Binario ya actualizado por fuera
- **DADO** un dashboard en ejecución con la versión A
- **Y** un binario instalado que ya es la última versión B
- **CUANDO** se consulta `GET /api/system/version`
- **ENTONCES** la respuesta trae `status: restart_required`, `installed: B` y `restart_required: true`
- **Y** el aviso de la interfaz ofrece «Reiniciar» en lugar de «Actualizar»
- **Y** `POST /api/system/update` relanza el dashboard con el binario instalado sin ejecutar `go install`

#### Scenario: Binario instalado también por detrás
- **DADO** un dashboard con la versión A y un binario instalado B, distinto de A y anterior a la última versión C
- **CUANDO** se consulta `GET /api/system/version`
- **ENTONCES** el estado es `update_available` con `restart_required: true`
- **Y** la actualización instala C y después relanza el dashboard

### Requirement: Reinicio forzado con la última versión (RF-UPD-10)

El dashboard DEBE ofrecer en todo momento la acción «Reiniciar con la última versión» (`POST /api/system/restart`, solo desde el propio dashboard). Esa acción DEBE instalar la última versión publicada cuando el binario instalado esté por detrás y DEBE relanzar siempre el dashboard con el binario instalado, también cuando ya está al día. Si la instalación falla, el dashboard en ejecución DEBE seguir sirviendo y mostrar el motivo con la orden manual del canal en uso: `@main` en el canal beta y `@latest` en el estable.

#### Scenario: Reinicio forzado sin versión nueva
- **DADO** un dashboard al día
- **CUANDO** el usuario pulsa «Reiniciar con la última versión»
- **ENTONCES** no se instala nada
- **Y** el dashboard se relanza en el mismo puerto y la página se recarga cuando responde el proceso nuevo, aunque la versión sea la misma

#### Scenario: Reinicio forzado con versión nueva
- **DADO** un binario instalado por detrás de la última versión
- **CUANDO** el usuario pulsa «Reiniciar con la última versión»
- **ENTONCES** se instala la última versión y el dashboard se relanza con ella

### Requirement: Instalación propia en Windows independiente del nombre apartado (RF-UPD-11)

En Windows, la actualización del propio binario con `go install` DEBE compilar en un directorio temporal dentro del destino y sustituir el binario apartando el anterior con un nombre único, de modo que un `axiom.exe~` ocupado por un proceso anterior no impida la actualización. Si la sustitución falla, el binario anterior DEBE quedar restaurado en su ruta.

#### Scenario: Nombre apartado ocupado por un dashboard anterior
- **DADO** un `axiom.exe~` en uso por un proceso anterior en el directorio de destino
- **CUANDO** se actualiza Axiom con `go install` en Windows
- **ENTONCES** el binario nuevo queda en `axiom.exe`
- **Y** el anterior queda apartado con un nombre único terminado en `.old`

#### Scenario: Falla la sustitución
- **DADO** una compilación correcta en el directorio temporal
- **CUANDO** no se puede mover el binario nuevo a su ruta
- **ENTONCES** el binario anterior vuelve a su ruta y la actualización informa del fallo

