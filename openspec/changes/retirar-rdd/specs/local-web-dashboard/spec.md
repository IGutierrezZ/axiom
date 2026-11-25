# Delta for local-web-dashboard

## ADDED Requirements

### Requirement: El Dashboard No Expone el Modo de Revisión de RDD (REQ-46.6)

El servidor del Dashboard NO DEBE exponer el endpoint `/api/project/review-mode` ni ningún otro de modo de revisión, y la interfaz web NO DEBE incluir el selector `reviewmode.js`. Una petición a esa ruta DEBE responder como ruta inexistente (404). El resto de endpoints del Dashboard NO cambia.

#### Scenario: El endpoint de modo de revisión no existe

- **DADO** el servidor del Dashboard en marcha
- **CUANDO** se solicita `GET /api/project/review-mode`
- **ENTONCES** la respuesta es 404

#### Scenario: La interfaz no carga el selector

- **DADO** los recursos estáticos del Dashboard
- **CUANDO** se listan
- **ENTONCES** no existe `reviewmode.js` ni ninguna referencia a él en las páginas
