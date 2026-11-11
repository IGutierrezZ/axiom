// Lógica pura de la pantalla de modelos por fase (Ecosistema): qué opciones ofrece cada
// adaptador, cómo se validan en el navegador las filas editadas y cómo se arma el parche
// que se envía a POST /api/ecosystem/models. Sin acceso al DOM, para poder probarla con
// Node.
//
// El servidor repite estas reglas con el mismo validador que `axiom sync` y es quien
// decide; aquí solo se adelantan los avisos para no esperar a la respuesta.
//
// Un «bloque» es un elemento de `adapters` de GET /api/ecosystem/models:
//   { id, name, phases, assignments: { fase: { model, effort } }, defaults, options, ... }
// Una «fila» editada es { phase, model, effort } con los textos tal como están en los campos.

export const ADAPTER_CLAUDE = 'claude-code';
export const ADAPTER_OPENCODE = 'opencode';
export const ADAPTER_COPILOT = 'copilot';

// Un carácter de control en el modelo o el esfuerzo acabaría en el frontmatter del agente:
// el servidor lo rechaza y aquí se avisa antes.
const CONTROL = /[\u0000-\u001f\u007f]/;

// splitModelSpec separa «proveedor/modelo» por la primera barra (o dos puntos), igual que
// model.SplitModelSpec del servidor. Devuelve null si no tiene las dos partes.
export function splitModelSpec(spec) {
  const text = String(spec ?? '').trim();
  const at = text.search(/[/:]/);
  if (at <= 0 || at === text.length - 1) return null;
  return { provider: text.slice(0, at), model: text.slice(at + 1) };
}

// catalogModel busca en el catálogo de OpenCode del bloque el modelo «proveedor/modelo».
// Devuelve null si no hay catálogo o el modelo no está en él.
function catalogModel(block, spec) {
  const parts = splitModelSpec(spec);
  if (!parts) return null;
  const provider = (block?.options?.providers || []).find(p => p.id === parts.provider);
  return provider?.models?.find(m => m.id === parts.model) || null;
}

// hasCatalog dice si el bloque trae un catálogo de OpenCode con proveedores.
export function hasCatalog(block) {
  return Array.isArray(block?.options?.providers) && block.options.providers.length > 0;
}

// modelInput describe el control del modelo: un selector de alias (Claude Code), un
// selector agrupado por proveedor (OpenCode con catálogo) o un texto libre con sugerencias
// (Copilot, y OpenCode si no hay catálogo).
//   { kind: 'select', options: [{ value, label }] }
//   { kind: 'grouped', groups: [{ label, options: [{ value, label }] }] }
//   { kind: 'text', suggestions: [string] }
export function modelInput(block) {
  if (block?.id === ADAPTER_CLAUDE) {
    return { kind: 'select', options: (block.options?.models || []).map(m => ({ value: m.id, label: m.id })) };
  }
  if (block?.id === ADAPTER_OPENCODE && hasCatalog(block)) {
    return {
      kind: 'grouped',
      groups: block.options.providers.map(p => ({
        label: p.name || p.id,
        options: (p.models || []).map(m => ({ value: `${p.id}/${m.id}`, label: m.name || m.id }))
      }))
    };
  }
  return { kind: 'text', suggestions: block?.options?.suggestions || [] };
}

// effortInput describe el control del esfuerzo para el modelo elegido: un selector con los
// valores válidos cuando se conocen (Claude Code por alias; OpenCode por variantes del
// catálogo) o texto libre cuando no (Copilot, OpenCode sin catálogo).
//   { kind: 'select', values: [string] }  (el vacío «por defecto» lo añade quien pinta)
//   { kind: 'text' }
//   { kind: 'none' }  sin modelo no hay esfuerzo que elegir
export function effortInput(block, model) {
  const value = String(model ?? '').trim();
  if (!value) return { kind: 'none' };
  if (block?.id === ADAPTER_CLAUDE) {
    const option = (block.options?.models || []).find(m => m.id === value);
    return { kind: 'select', values: option ? option.efforts || [] : [] };
  }
  if (block?.id === ADAPTER_OPENCODE && hasCatalog(block)) {
    const found = catalogModel(block, value);
    return { kind: 'select', values: found ? found.variants || [] : [] };
  }
  return { kind: 'text' };
}

// defaultLabel es el texto de la opción vacía del selector de modelo: lo que pasa si la
// fase no tiene asignación.
export function defaultLabel(block, phase) {
  const fallback = block?.defaults?.[phase]?.model;
  if (fallback) return `por defecto de Axiom (${fallback})`;
  return block?.id === ADAPTER_COPILOT ? 'hereda el modelo de la conversación' : 'sin asignar (hereda)';
}

// rowsFor convierte las asignaciones guardadas del bloque en filas, en el orden de sus fases.
export function rowsFor(block) {
  return (block?.phases || []).map(phase => {
    const saved = block.assignments?.[phase];
    return { phase, model: saved?.model || '', effort: saved?.effort || '' };
  });
}

function clean(row) {
  return { phase: row.phase, model: String(row.model ?? '').trim(), effort: String(row.effort ?? '').trim() };
}

// rowProblem explica por qué una fila no es admisible, o devuelve '' si lo es.
function rowProblem(block, row) {
  const { phase, model, effort } = clean(row);
  if (!model) return effort ? `${phase}: elige un modelo antes de indicar el esfuerzo` : '';
  if (CONTROL.test(model) || CONTROL.test(effort)) return `${phase}: el modelo y el esfuerzo no pueden llevar saltos de línea ni caracteres de control`;
  if (block.id === ADAPTER_CLAUDE) {
    const option = (block.options?.models || []).find(m => m.id === model);
    if (!option) return `${phase}: el modelo «${model}» no es válido (opciones: ${(block.options?.models || []).map(m => m.id).join(', ')})`;
    if (effort && !(option.efforts || []).includes(effort)) {
      return (option.efforts || []).length
        ? `${phase}: el esfuerzo «${effort}» no es válido para ${model} (opciones: ${option.efforts.join(', ')})`
        : `${phase}: ${model} no admite esfuerzo`;
    }
    return '';
  }
  if (block.id === ADAPTER_OPENCODE) {
    if (!splitModelSpec(model)) return `${phase}: el modelo «${model}» debe tener el formato proveedor/modelo`;
    if (hasCatalog(block)) {
      const found = catalogModel(block, model);
      if (!found) return `${phase}: el modelo «${model}» no está en el catálogo de OpenCode`;
      if (effort && !(found.variants || []).includes(effort)) {
        return (found.variants || []).length
          ? `${phase}: la variante «${effort}» no es válida para ${model} (opciones: ${found.variants.join(', ')})`
          : `${phase}: ${model} no admite variantes`;
      }
    }
  }
  return '';
}

// validateRows devuelve todas las violaciones de las filas editadas, una por fila.
export function validateRows(block, rows) {
  const violations = [];
  for (const row of rows || []) {
    const problem = rowProblem(block, row);
    if (problem) violations.push(problem);
  }
  return violations;
}

// buildPatch arma el cuerpo del POST con solo las fases que cambian respecto a lo guardado.
// Una fila sin modelo que tenía asignación se envía con modelo vacío (quitarla). Si no hay
// cambios, `patch` queda vacío. Devuelve { body, changed, violations }; con violaciones no
// hay que enviar nada.
export function buildPatch(block, rows) {
  const violations = validateRows(block, rows);
  const assignments = {};
  for (const raw of rows || []) {
    const row = clean(raw);
    const saved = block?.assignments?.[row.phase];
    const savedModel = saved?.model || '';
    const savedEffort = saved?.effort || '';
    if (row.model === savedModel && (row.model === '' || row.effort === savedEffort)) continue;
    assignments[row.phase] = row.model ? { model: row.model, effort: row.effort } : { model: '' };
  }
  const changed = Object.keys(assignments);
  return { body: { adapter: block?.id, assignments }, changed, violations };
}

// shortList abrevia una lista de ficheros para un mensaje: los primeros y «y N más».
function shortList(items, max = 4) {
  if (items.length <= max) return items.join(', ');
  return `${items.slice(0, max).join(', ')} y ${items.length - max} más`;
}

// savedMessage resume una respuesta correcta del POST: qué ficheros se regeneraron o, si
// no hubo ninguno, que el cambio quedó guardado en el estado del proyecto.
export function savedMessage(response) {
  const files = Array.isArray(response?.regenerated) ? response.regenerated : [];
  if (files.length === 0) {
    return 'Guardado en el estado del proyecto. No se ha regenerado ningún fichero desde aquí.';
  }
  const noun = files.length === 1 ? 'fichero regenerado' : 'ficheros regenerados';
  return `Guardado. ${files.length} ${noun}: ${shortList(files)}.`;
}

// noChangesMessage es el aviso cuando se pulsa «Guardar» sin haber cambiado nada.
export const NO_CHANGES_MESSAGE = 'No hay cambios que guardar.';

// withSavedAssignments devuelve el bloque con las asignaciones que devolvió el servidor,
// para repintarlo sin volver a pedirlo todo.
export function withSavedAssignments(block, response) {
  return { ...block, assignments: response?.assignments || {} };
}
