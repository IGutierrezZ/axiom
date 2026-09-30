# ODD: Modernización Visual de Axiom UI — Adopción del Sistema de Diseño Axiom (Modern Light Theme & Iconografía Vectorial)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/ui-modern-light-theme.md`.  
> Rama: `feat/ui-modern-light-theme` en worktree `C:\repos\axiom-wt\ui-modern-light-theme`.

## Objetivo

Portar a la consola web de `axiom ui` (`internal/dashboard/assets/`) los avances visuales y de diseño desarrollados en `app-knowledge-agent` bajo la especificación canónica del Sistema de Diseño de Axiom:
1. **Adopción del Modern Light Theme:** Paleta de alta claridad y contraste (superficies Slate/Zinc `#f8fafc` / `#ffffff`, acentos Sky/Deep Slate `#0284c7`, bordes sutiles `#e2e8f0` / `#cbd5e1`, sombras suaves y elevaciones elegantes).
2. **Erradicación Total de Emojis:** Sustitución de emojis de sistema dispersos en encabezados, botones, tarjetas, pestañas y modales por iconografía vectorial SVG nítida y accesible con trazo profesional.
3. **Refinamiento de Componentes UI:** Botones con variantes tipadas, tarjetas con hover dinámico, badges semánticos con contraste accesible, controles de formulario estilizados y consolas de código monoespaciadas.
4. **Preservación Funcional y Retrocompatibilidad:** Mantener el 100% de la lógica de negocio, endpoints REST, selector de carpetas y suite de pruebas en verde.

---

## Tareas

- [x] **T1 · Tokens Centralizados y Paleta Semántica (`internal/dashboard/assets/style.css`)**
  - Implementar variables `:root` de Modern Light Theme y alias para selectores existentes.
- [x] **T2 · Iconografía Vectorial SVG y Limpieza en `index.html`**
  - Sustituir emojis en la cabecera, selector de proyectos, pestañas de navegación y modales por SVGs vectoriales.
- [x] **T3 · Renderizado Dinámico e Insignias en `app.js`**
  - Reemplazar emojis y caracteres de texto en generación de tarjetas, estados de barrera, badges de bugs/features y feedback de diagnósticos con diccionario SVG `ICONS`.
- [x] **T4 · Refinamiento Estético de Tarjetas, Tablas, Modales y Consolas (`style.css`)**
  - Ajustar estilos de tarjetas de incrementos, visor de handoffs, buzón de skills, modal de selector de carpetas y consolas de terminal.
- [x] **T5 · Verificación Automatizada y Compilación**
  - Ejecutar suite de pruebas de dashboard (`go test -v -run TestEcosystemUpgradePresentsBothPhases ./internal/dashboard`), compilar y verificar.

---

## Verificación Ejecutable
- `go test ./internal/dashboard/... -v`
- `go build ./cmd/axiom`
