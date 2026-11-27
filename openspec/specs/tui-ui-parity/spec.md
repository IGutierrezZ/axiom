# Especificación Viva: Gestión del Ecosistema desde la Web UI (antes, paridad entre TUI y Web UI)

> **Dominio:** `tui-ui-parity`  
> **Versión Canónica:** 1.0.0  
> **Estado:** Vigente  
> **Idioma:** Español (Castellano peninsular)

---

## 1. Capacidad: `dashboard-ecosystem-management`

Permite a los usuarios operar las tareas de mantenimiento, diagnóstico y configuración del ecosistema de desarrollo de Axiom directamente desde el Dashboard Web SPA (`axiom ui`).

### Requirement: Diagnóstico de Salud del Sistema (Axiom Doctor) en Web UI (RF-UIP-01)
El servidor HTTP local DEBE exponer el endpoint `GET /api/ecosystem/doctor` que recopila y retorna el estado de salud de los agentes compatibles (Claude Code, OpenCode, Codex, Pi), herramientas del sistema (Git, Go, Node.js), permisos del directorio de usuario (`~/.axiom/`) y configuración general. El frontend web DEBE renderizar estos diagnósticos en tarjetas con indicadores visuales claros (verde: saludable, amarillo: advertencia, rojo: error).
(Previously: la lista de agentes incluía Kiro, retirado por C-08-d.)

#### Scenario: Consulta de diagnóstico de salud del sistema
- **DADO** el servidor del Dashboard Web activo
- **CUANDO** se realiza una petición `GET /api/ecosystem/doctor`
- **ENTONCES** el servidor responde con código `200 OK`
- **Y** el cuerpo JSON contiene la lista de diagnósticos con campos `name`, `status` (`ok`, `warning`, `error`), `details` y `recommendation`

### Requirement: Sincronización y Actualización Reactiva de Herramientas en Web UI (RF-UIP-02)

El servidor HTTP local DEBE exponer los endpoints `POST /api/ecosystem/sync` y `POST /api/ecosystem/upgrade`. El endpoint `POST /api/ecosystem/upgrade` DEBE ejecutar secuencialmente la cadena `upgrade` ➔ `sync` en segundo plano, con reporte estructurado por fases (`phases.upgrade`, `phases.sync`) disponible en el estado del trabajo, omitiendo `sync` de forma segura si `upgrade` exige reinicio del proceso (`restart_required: true`). El frontend web DEBE ofrecer botones de acción interactiva en la pestaña `⚙️ Ecosistema & Herramientas`, seguir el avance del trabajo y mostrar un registro de salida con los resultados detallados de la operación.

#### Scenario: Ejecución de sincronización de configuraciones
- **DADO** la pestaña de Ecosistema en el Dashboard Web
- **CUANDO** el usuario pulsa el botón "Sincronizar Configuraciones"
- **Y** se envía una petición `POST /api/ecosystem/sync`
- **ENTONCES** el servidor ejecuta la sincronización de archivos gestionados y reglas de agentes
- **Y** responde con código `200 OK` y el resumen de archivos sincronizados en formato JSON

#### Scenario: Ejecución de actualización encadenada a sincronización
- **DADO** la pestaña de Ecosistema en el Dashboard Web
- **CUANDO** el usuario pulsa el botón "Actualizar Herramientas"
- **Y** se envía una petición `POST /api/ecosystem/upgrade`
- **ENTONCES** el servidor responde `202 Accepted` y, en segundo plano, comprueba versiones, actualiza binarios y encadena la sincronización de agentes si no se requiere reinicio
- **Y** la interfaz muestra el informe consolidado por fases cuando el trabajo termina

### Requirement: Gestión Visual de Respaldos en Web UI (RF-UIP-03)
El servidor HTTP local DEBE exponer los endpoints `GET /api/ecosystem/backups`, `POST /api/ecosystem/backups/create` y `POST /api/ecosystem/backups/restore` integrados con el paquete `internal/backup`. El frontend web DEBE presentar la lista cronológica de snapshots en `~/.axiom/backups/`, permitir la creación de un nuevo respaldo bajo demanda con nombre descriptivo opcional y permitir restaurar un respaldo seleccionado con confirmación previa.

#### Scenario: Listado y creación de un respaldo desde el navegador
- **DADO** el servidor activo con respaldos existentes en `~/.axiom/backups/`
- **CUANDO** se realiza una petición `GET /api/ecosystem/backups`
- **ENTONCES** el servidor retorna la lista de respaldos con fecha, identificador y lista de archivos
- **CUANDO** se envía `POST /api/ecosystem/backups/create` con `{ "description": "pre-migracion" }`
- **ENTONCES** el servidor crea el nuevo snapshot y responde con `201 Created`

---

### Requirement: Inspección y Configuración de Modelos de IA en Web UI (RF-UIP-04)
El servidor HTTP local DEBE exponer el endpoint `GET /api/ecosystem/models` para consultar la asignación vigente de modelos de IA y configuraciones por rol para los agentes soportados.

#### Scenario: Consulta de modelos asignados en el Dashboard
- **DADO** una configuración activa de modelos en el proyecto o usuario
- **CUANDO** se realiza una petición `GET /api/ecosystem/models`
- **ENTONCES** el servidor responde con `200 OK` y el mapa de asignaciones de modelos y niveles de esfuerzo de razonamiento

---

## 2. Capacidad: `tui-hub-and-sdd-governance`

Permite a los desarrolladores operar la gobernanza Spec-Driven Development y la gestión multi-proyecto directamente desde la interfaz interactiva de terminal Bubbletea.

