# Especificación de Requerimientos: Declaración de la Baseline del Proyecto y Guarda contra una Segunda Baseline (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Rebanada:** B (declaración y guardas) · **Tipo:** capacidad nueva
> **Idioma:** Español (castellano peninsular)

## Propósito

Definir cómo declara un proyecto, en `axiom.yaml`, dónde está su *baseline* (la especificación viva que archive mantiene) y con qué formato; cómo se valida esa declaración, incluida su compatibilidad con el almacén de artefactos; cómo se proyecta en el status v2 sin romper su contrato; y cómo se garantiza que ningún creador de Axiom produce una segunda baseline cuando la declarada es del proyecto. Sin declaración, la baseline es OpenSpec (`<planning home>/specs/`) y todo el comportamiento es idéntico al actual.

La documentación del proyecto que solo se usa como entrada y nunca se modifica NO se declara aquí: se declara como contexto de referencia en la capacidad `project-technical-context`.

## Términos

| Término | Definición |
|---|---|
| Baseline | Conjunto de documentos que archive actualiza al integrar los cambios (la especificación viva). |
| Baseline por defecto | OpenSpec: `<planning home>/specs/<dominio>/spec.md`, con requisitos de encabezado `Requirement:`. |
| Baseline del proyecto | La declarada en `baseline.documents` de `axiom.yaml`, en la ubicación y el formato del proyecto. |
| Documento con perfil de requisito | Documento direccionable por requisito: archive lo integra por composición (capacidad `spec-baseline-integration`). |
| Documento de prosa | Documento sin perfil de requisito: archive lo integra por edición dirigida o solo lo registra. |
| Creador | Operación de Axiom que crea o actualiza `openspec/specs/` u `openspec/INDEX.md`. |
| Almacén de ficheros | Almacén de artefactos `openspec` o `hybrid` (`artifact_store` de `openspec/config.yaml`): el cambio vive como ficheros en su carpeta de `openspec/changes/`. Con `engram` o `none` no existe esa carpeta. |

## Ejemplo de declaración (no normativo)

```yaml
baseline:
  documents:
    - domain: requisitos-funcionales
      path: 01_Requisitos_Funcionales.md
      scenarios: omit
      requirement:
        heading: '^### (?P<id>RF-\d+): (?P<title>.+)$'
        level: 3
        identity: id
        render: '### {id}: {title}'
    - domain: modelo-de-datos
      path: 03_Modelo_Datos.md
      integration: directed
    - domain: flujos-de-negocio
      path: 04_Flujos_Negocio.md
      integration: register
```

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| REQ-24.13 | Bloque `baseline:` opcional y defecto OpenSpec | Sin bloque, comportamiento actual |
| REQ-24.14 | Documentos por dominio | Un dominio, un documento; rutas contenidas |
| REQ-24.15 | Perfil de requisito | Patrón con `id` y `title`, nivel, identidad y plantilla |
| REQ-24.16 | Documentos de prosa y modo de integración | `directed` o `register`, nunca ambos con perfil |
| REQ-24.17 | Validación de la declaración | Loader y `axiom workspace validate` (también el almacén, REQ-24.62), en español, código 1 |
| REQ-24.18 | Una declaración inválida falla cerrada | Nunca degrada a la baseline por defecto; con un puntero ilegible rechaza antes (REQ-24.7) |
| REQ-24.19 | Proyección de `baseline` en el status v2 | Localizadores aditivos, sin parsear YAML |
| REQ-24.20 | Contrato del status v2 solo aditivo | Ninguna clave existente cambia |
| REQ-24.21 | Frontera de configuración | Estructura en `axiom.yaml`, conducta en `openspec/config.yaml` |
| REQ-24.22 | Consulta única de baseline del proyecto | Todos los creadores responden igual |
| REQ-24.23 | Guarda en CLI, dashboard y TUI | Ninguna escritura crea `openspec/specs/` ni `INDEX.md` |
| REQ-24.24 | Guarda en las skills | `sdd-init` y la rama sin baseline de `sdd-archive` |
| REQ-24.25 | Lecturas de catálogo | Informan de que la baseline es externa |
| REQ-24.60 | Clave `scenarios` por documento | `include` por defecto (exige `level` ≤ 3) u `omit`; solo con perfil de requisito |
| REQ-24.62 | Almacén de ficheros para baseline del proyecto y contexto mantenido | `engram` y `none` se reportan y detienen el archivado |

## No cubre esta capacidad

- La migración de la baseline de un proyecto existente.
- Indexar la baseline del proyecto en livingdoc, dashboard o knowledge: solo se informa de que es externa (REQ-24.25).
- Las correcciones del arranque en frío de knowledge (nivel `####` del grabador frente a `###` del compose, dominios falsos del barrido, sobrescritura de `coldstart`), salvo las guardas de esta capacidad.

---

## 1. Capacidad: `spec-baseline-declaration`

El bloque `baseline:` de `axiom.yaml` declara los documentos de la especificación viva del proyecto. Su ausencia significa baseline OpenSpec. Con la declaración presente, Axiom lee y escribe esos documentos y ninguna operación crea una segunda baseline.

