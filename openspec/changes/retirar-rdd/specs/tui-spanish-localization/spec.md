# Delta for tui-spanish-localization

## MODIFIED Requirements

### Requirement: Localización al Castellano del Menú de Bienvenida (REQ-16.2)
El menú principal de bienvenida (`WelcomeOptions`) y sus microtextos auxiliares deben presentarse íntegramente en español (castellano peninsular).
(Previously: la lista de etiquetas incluía «Reiniciar almacén de revisiones» y «Revisión formal RDD»; ambas entradas del menú se retiran con RDD (C-01-e).)

#### Scenario: Visualización de opciones en español
- **DADO QUE** el usuario se encuentra en la pantalla de bienvenida
- **CUANDO** se renderiza la lista de opciones
- **ENTONCES** las etiquetas deben ser:
  - "Iniciar instalación"
  - "Actualizar herramientas" / "Actualizar herramientas (al día)" / "Actualizar herramientas ★"
  - "Sincronizar configuraciones"
  - "Actualizar y sincronizar"
  - "Configurar modelos"
  - "Crear agente personalizado"
  - "Plugins comunitarios de OpenCode"
  - "Desinstalar plugin de OpenCode"
  - "Perfiles SDD de OpenCode"
  - "Gestionar respaldos"
  - "Desinstalación gestionada"
  - "Herramientas y plugins comunitarios"
  - "Salir"

#### Scenario: Microtextos de navegación y encabezados
- **DADO QUE** se visualiza el frame del menú
- **CUANDO** se renderizan los textos fijos
- **ENTONCES** la cabecera de la lista debe mostrar "Menú"
- **Y** la barra inferior de ayuda debe mostrar `j/k: navegar • enter: seleccionar • q: salir`.
