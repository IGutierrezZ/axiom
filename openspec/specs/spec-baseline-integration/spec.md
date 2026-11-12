# Especificación de Requerimientos: Integración de los Cambios en la Baseline del Proyecto (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Rebanada:** C (integración) · **Tipo:** capacidad nueva
> **Idioma:** Español (castellano peninsular)

## Propósito

Definir cómo integra archive un cambio en la baseline declarada del proyecto: la composición determinista sobre documentos direccionables por requisito (patrón de encabezado, identidad por ID, plantilla de render, duplicados, frontera de bloque, fidelidad de bytes), la integración de documentos de prosa mediante edición dirigida con verificación nativa y traza por entrada, las precondiciones de la integración y la lectura de la baseline por las fases de planificación.

El mecanismo de edición dirigida con verificación nativa y traza (REQ-24.50 a REQ-24.52) se especifica aquí una sola vez; la integración del delta de contexto técnico lo reutiliza (`sdd-archive-lifecycle`, REQ-24.55).

**Resultado esperado:** archive integra el cambio en los documentos declarados sin alterar un solo byte ajeno a los bloques editados, rechaza toda operación ambigua antes de escribir y deja una traza verificable por entrada.

## Términos

| Término | Definición |
|---|---|
| Bloque de requisito | Desde la línea de encabezado de un requisito hasta, sin incluirla, la siguiente línea de encabezado ATX de nivel menor o igual que el del requisito, sea requisito o no, fuera de bloques de código. Los encabezados de nivel mayor (por ejemplo h4 bajo un requisito h3) pertenecen al bloque. |
| Cola de secciones contiguas | Secciones no-requisito del mismo nivel que siguen directamente a un bloque, hasta el siguiente requisito de ese nivel o el siguiente encabezado de nivel menor. No pertenecen al bloque. |
| Familia de IDs | IDs que comparten el prefijo no numérico (por ejemplo `RF-`). |
| ID provisional | ID del encabezado de un requisito ADDED de las delta specs del cambio con identidad `id`. Archive lo sustituye por el ID definitivo en todo el texto que integra (REQ-24.59). |
| Referencia a un ID provisional | Aparición exacta del ID provisional como token completo en el texto a integrar: ni precedida ni seguida por letras, dígitos, `_` ni `-`, con distinción de mayúsculas y minúsculas. Todo token igual a un ID provisional del cambio es una referencia a él. |
| Texto integrado | Texto que archive escribe en los documentos del proyecto a partir del cambio: encabezados y cuerpos de los bloques de requisito, la nota de motivo que `--supersede` integra en un bloque conservado, las secciones del cambio en documentos de prosa y el texto de las entradas del delta de contexto. No incluye el texto preexistente de los documentos ni los ficheros del propio cambio. |
| Entrada | Cada requisito de la delta spec de un dominio de prosa o cada entrada del delta de contexto. |
| Traza de integración | Registro por entrada de lo que archive integró, en la carpeta del cambio y en el informe de archive. |
| Comprobación previa | Operación de solo lectura (`axiom sdd integrate check`) que aplica a las delta specs del cambio las validaciones de contenido que archive aplica antes de escribir (REQ-24.46), sin escribir nada y sin las precondiciones de git (REQ-24.49) ni la autorización de rutas externas (REQ-24.37). |

## Ejemplo de traza (no normativo)

| Identificador | Documento | Ancla | Acción | Estado |
|---|---|---|---|---|
| RF-9001 → RF-008 | 01_Requisitos_Funcionales.md | RF-008 | added | integrated |
| inc-99-ejemplo | 03_Modelo_Datos.md | inc-99-ejemplo | modified | integrated |
| inc-99-ejemplo | 04_Flujos_Negocio.md | inc-99-ejemplo | added | pending |

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| REQ-24.38 | Composición nativa intacta sin perfil | Byte a byte como hasta ahora |
| REQ-24.39 | Reconocimiento de requisitos e identidad | Por patrón, nivel e ID o título |
| REQ-24.40 | Rechazo de IDs duplicados | Operación ambigua rechazada, resto operable |
| REQ-24.41 | Frontera de bloque por nivel | Subsecciones dentro; contiguas fuera, con aviso |
| REQ-24.42 | Formato del delta, render y frontera del cuerpo | Delta nativo, encabezado por plantilla, cuerpo íntegro sin encabezados que cierren el bloque, con detección temprana |
| REQ-24.43 | ADDED con ubicación explícita | ID provisional, definitivo asignado en archive |
| REQ-24.44 | MODIFIED, REMOVED y RENAMED por ID | Cambiar el título es MODIFIED; no se renumera |
| REQ-24.45 | Fidelidad de bytes | LF, CRLF y BOM preservados |
| REQ-24.46 | Validación completa antes de escribir | Todo o nada |
| REQ-24.47 | `axiom sdd archive-compose` con `--cwd` y `--document` | Documento y perfil desde `axiom.yaml` |
| REQ-24.48 | Dominio no declarado | Rechazado; nunca crea `openspec/specs/<dominio>` |
| REQ-24.49 | Precondiciones de la integración | Git y árbol limpio; registros del proyecto intactos |
| REQ-24.50 | Integración dirigida de documentos de prosa | Sección del cambio por edición dirigida |
| REQ-24.51 | Verificación nativa | Presencia, anclas y alcance por diff |
| REQ-24.52 | Traza de integración | Por entrada, en el cambio y en el informe |
| REQ-24.53 | Modo solo registro | Entradas pendientes sin editar el documento |
| REQ-24.54 | Lectura de la baseline por las fases | Localizador del status, sin rutas fijas; `sdd-spec` ejecuta la comprobación previa |
| REQ-24.59 | Sustitución de los IDs provisionales | Todo el texto integrado usa los IDs definitivos; correspondencia única y en la traza |
| REQ-24.61 | Escenarios `include` u `omit` | El bloque del proyecto no impone el formato de escenarios si el documento lo declara |

## No cubre esta capacidad

- Actualizar `REGISTRO_INCREMENTOS.md` ni `REGISTRO_BUGS.md` del proyecto.
- Evaluar la corrección semántica de una edición en prosa: la verificación es determinista y solo comprueba presencia y alcance.
- Renumerar requisitos existentes.
- Normalizar los IDs duplicados de un documento: se exige previamente en el repositorio del proyecto.
- Elegir el actor que ejecuta la edición dirigida (decisión de diseño).

---

## 1. Capacidad: `spec-baseline-integration`

La composición generaliza la nativa de OpenSpec sin sustituirla: con perfil de requisito aplica identidad por ID y plantilla de render; sin perfil, conserva el comportamiento actual byte a byte. Los documentos de prosa se integran con edición dirigida y verificación nativa.

### Requirement: Consulta de un Requisito Existente en la Baseline para el Bug sin Delta (REQ-29.14)

El motor DEBE poder resolver si un identificador de requisito existe en la baseline (`openspec/specs/`, la spec viva del proyecto) y, si existe, devolver sus escenarios. La resolución se hace por el identificador `REQ-<n>.<m>` que contiene el nombre del requisito (REQ-29.3 de `sdd-iso-evidence`), y un requisito cuyo nombre no contiene ese identificador no puede citarse (supuesto visible). La consulta es de solo lectura y NO DEBE modificar ningún documento. La usa el bug con requisito existente (REQ-29.5 de `sdd-iso-evidence`) para comprobar, en un cambio ISO, que el `Cubre:` apunta a un requisito existente y para conocer los escenarios que la matriz de verify debe cubrir (REQ-29.6). Un identificador que no existe se informa como tal, sin error de ejecución, y un requisito existente sin escenarios se informa como existente con una lista de escenarios vacía.

#### Scenario: Un identificador existente se resuelve con sus escenarios

- **DADO** la baseline con el requisito `Subcomando axiom sdd status (REQ-13.1)` con dos escenarios
- **CUANDO** se consulta `REQ-13.1`
- **ENTONCES** se informa de que existe y se devuelven sus dos escenarios

#### Scenario: Un identificador inexistente se informa sin error

- **DADO** la misma baseline y el identificador `REQ-13.99`
- **CUANDO** se consulta `REQ-13.99`
- **ENTONCES** se informa de que no existe
- **Y** la consulta no falla ni modifica ningún documento

#### Scenario: Un requisito sin identificador en su nombre no puede citarse

- **DADO** la baseline con un requisito cuyo nombre no contiene `REQ-<n>.<m>`
- **CUANDO** se consulta cualquier identificador
- **ENTONCES** ese requisito no se devuelve para ningún identificador

#### Scenario: Un requisito existente sin escenarios se informa como tal

- **DADO** un requisito existente sin ningún `#### Scenario:`
- **CUANDO** se consulta su identificador
- **ENTONCES** se informa de que existe con una lista de escenarios vacía

#### Scenario: La consulta no escribe

- **DADO** cualquier consulta
- **CUANDO** termina
- **ENTONCES** el árbol de trabajo no presenta ningún cambio

