// Modelo por fase (Ecosistema): una tabla editable por adaptador instalado en el proyecto
// abierto (OpenCode, Claude Code y GitHub Copilot). Cada fila es una fase SDD con su modelo
// y su esfuerzo; «Guardar» envía solo lo que cambió a POST /api/ecosystem/models, que lo
// guarda en .axiom/state.json y regenera los ficheros del adaptador en el proyecto.
//
// La lógica (opciones, validación, parche) vive en core/phasemodels.js, sin DOM y probada con
// Node; aquí solo se pinta y se cablean los eventos. El servidor repite la validación y es
// quien decide.

import { api, escapeHtml } from '../core/api.js';
import { setBusy, stateEmpty, tableWrap, toast } from '../core/components.js';
import {
  NO_CHANGES_MESSAGE, buildPatch, defaultLabel, effortInput, modelInput, rowsFor, savedMessage, withSavedAssignments
} from '../core/phasemodels.js';
import { violationsFromError } from '../core/projectform.js';

// Bloques pintados, por adaptador: se actualizan con la respuesta de cada guardado para
// comparar la siguiente edición con lo que ya está guardado.
const blocks = new Map();

const INTRO = 'Modelo y esfuerzo de cada sub-agente SDD del proyecto abierto. Se guardan en el estado del proyecto (.axiom/state.json) y se regeneran los ficheros del adaptador. Una fase sin modelo usa el valor por defecto del adaptador.';

function option(value, label, selected) {
  return `<option value="${escapeHtml(value)}"${selected ? ' selected' : ''}>${escapeHtml(label)}</option>`;
}

// modelControlHtml pinta el control del modelo de una fila según el tipo de adaptador.
function modelControlHtml(block, row) {
  const input = modelInput(block);
  const label = `aria-label="Modelo de ${escapeHtml(row.phase)}"`;
  const empty = option('', defaultLabel(block, row.phase), row.model === '');
  if (input.kind === 'text') {
    return `<input type="text" class="form-input" data-pm-model ${label} list="pm-dl-${escapeHtml(block.id)}" value="${escapeHtml(row.model)}" placeholder="${escapeHtml(defaultLabel(block, row.phase))}" autocomplete="off" spellcheck="false">`;
  }
  const known = new Set();
  let body;
  if (input.kind === 'grouped') {
    body = input.groups.map(group => {
      const items = group.options.map(o => { known.add(o.value); return option(o.value, o.label, o.value === row.model); }).join('');
      return `<optgroup label="${escapeHtml(group.label)}">${items}</optgroup>`;
    }).join('');
  } else {
    body = input.options.map(o => { known.add(o.value); return option(o.value, o.label, o.value === row.model); }).join('');
  }
  // Un valor guardado que ya no está entre las opciones se conserva visible, para no perderlo en silencio.
  const stale = row.model && !known.has(row.model) ? option(row.model, `${row.model} (no está en las opciones)`, true) : '';
  return `<select class="form-select" data-pm-model ${label}>${empty}${stale}${body}</select>`;
}

// effortControlHtml pinta el control del esfuerzo según el modelo elegido en la fila.
function effortControlHtml(block, phase, model, effort) {
  const input = effortInput(block, model);
  const label = `aria-label="Esfuerzo de ${escapeHtml(phase)}"`;
  if (input.kind === 'text') {
    return `<input type="text" class="form-input" data-pm-effort ${label} value="${escapeHtml(effort)}" placeholder="por defecto" autocomplete="off" spellcheck="false">`;
  }
  if (input.kind === 'none' || input.values.length === 0) {
    const hint = input.kind === 'none' ? 'elige primero un modelo' : 'sin esfuerzo para este modelo';
    return `<select class="form-select" data-pm-effort ${label} disabled>${option('', hint, true)}</select>`;
  }
  const stale = effort && !input.values.includes(effort) ? option(effort, `${effort} (no válido)`, true) : '';
  const items = input.values.map(v => option(v, v, v === effort)).join('');
  return `<select class="form-select" data-pm-effort ${label}>${option('', 'por defecto', effort === '')}${stale}${items}</select>`;
}

function rowHtml(block, row) {
  return `<tr data-phase="${escapeHtml(row.phase)}">
    <td class="mono">${escapeHtml(row.phase)}</td>
    <td>${modelControlHtml(block, row)}</td>
    <td data-pm-effort-cell>${effortControlHtml(block, row.phase, row.model, row.effort)}</td>
  </tr>`;
}

function sectionHtml(block) {
  const rows = rowsFor(block).map(row => rowHtml(block, row)).join('');
  const datalist = modelInput(block).kind === 'text'
    ? `<datalist id="pm-dl-${escapeHtml(block.id)}">${(block.options?.suggestions || []).map(s => `<option value="${escapeHtml(s)}"></option>`).join('')}</datalist>`
    : '';
  const dest = (block.destinations || []).map(d => `<code>${escapeHtml(d)}</code>`).join(' ');
  const catalogError = block.options?.catalog_error
    ? `<p class="eco-note text-bad" role="status">No se pudo cargar el catálogo de modelos de OpenCode (${escapeHtml(block.options.catalog_error)}): puedes escribir «proveedor/modelo» a mano y la variante, pero no se comprueban contra el catálogo.</p>`
    : '';
  const notes = (block.notes || []).map(n => `<p class="eco-note">${escapeHtml(n)}</p>`).join('');
  return `<section class="eco-models-adapter" data-adapter="${escapeHtml(block.id)}" aria-labelledby="pm-h-${escapeHtml(block.id)}">
    <div class="eco-models-head">
      <h3 class="eco-models-title" id="pm-h-${escapeHtml(block.id)}">${escapeHtml(block.name)}</h3>
      ${dest ? `<span class="eco-note">Se regenera en ${dest}</span>` : ''}
    </div>
    ${notes}${catalogError}
    ${tableWrap(`<table class="data-table eco-models-table"><thead><tr><th>Fase</th><th>Modelo</th><th>Esfuerzo / variante</th></tr></thead><tbody>${rows}</tbody></table>${datalist}`, `Modelos por fase de ${block.name}`)}
    <div class="eco-wt-actions">
      <button type="button" class="btn btn-primary" data-pm-save>Guardar modelos de ${escapeHtml(block.name)}</button>
      <span class="eco-note" data-pm-status role="status"></span>
    </div>
    <div class="alert alert-bad" data-pm-errors role="alert" hidden></div>
  </section>`;
}

// modelsHtml registra los bloques y devuelve el HTML de todas las tablas.
export function modelsHtml(adapters) {
  blocks.clear();
  const list = Array.isArray(adapters) ? adapters : [];
  if (list.length === 0) {
    return `<div class="eco-pad">${stateEmpty('Sin adaptadores con modelo por fase', 'El proyecto abierto no tiene instalados OpenCode, Claude Code ni GitHub Copilot. Instálalos con «axiom install --scope workspace --agent …» para elegir su modelo por fase.', 'cube')}</div>`;
  }
  for (const block of list) blocks.set(block.id, block);
  return `<div class="eco-pad"><p class="eco-note">${escapeHtml(INTRO)}</p>${list.map(sectionHtml).join('')}</div>`;
}

function readRows(section) {
  return [...section.querySelectorAll('tr[data-phase]')].map(tr => ({
    phase: tr.dataset.phase,
    model: tr.querySelector('[data-pm-model]').value,
    effort: tr.querySelector('[data-pm-effort]').value
  }));
}

function showErrors(section, violations) {
  const box = section.querySelector('[data-pm-errors]');
  box.textContent = '';
  if (!violations.length) {
    box.hidden = true;
    return;
  }
  const intro = document.createElement('p');
  intro.textContent = 'No se pueden guardar los modelos. Corrige estos errores:';
  const list = document.createElement('ul');
  for (const violation of violations) {
    const item = document.createElement('li');
    item.textContent = violation;
    list.appendChild(item);
  }
  const content = document.createElement('div');
  content.className = 'grow';
  content.append(intro, list);
  box.append(content);
  box.hidden = false;
}

function showStatus(section, text, tone = '') {
  const status = section.querySelector('[data-pm-status]');
  status.textContent = text;
  status.className = `eco-note${tone ? ` text-${tone}` : ''}`;
}

// onModelChange repinta el esfuerzo de la fila para el modelo elegido y conserva el valor si
// sigue siendo válido (o si el esfuerzo es de texto libre).
function onModelChange(section, tr) {
  const block = blocks.get(section.dataset.adapter);
  const model = tr.querySelector('[data-pm-model]').value;
  const cell = tr.querySelector('[data-pm-effort-cell]');
  const previous = cell.querySelector('[data-pm-effort]')?.value || '';
  const input = effortInput(block, model);
  const keep = input.kind === 'text' || (input.kind === 'select' && input.values.includes(previous)) ? previous : '';
  cell.innerHTML = effortControlHtml(block, tr.dataset.phase, model, keep);
}

async function save(section, button) {
  const block = blocks.get(section.dataset.adapter);
  const { body, changed, violations } = buildPatch(block, readRows(section));
  showErrors(section, violations);
  if (violations.length) {
    showStatus(section, '');
    return;
  }
  if (changed.length === 0) {
    showStatus(section, NO_CHANGES_MESSAGE);
    return;
  }
  const restore = setBusy(button, 'Guardando…');
  showStatus(section, '');
  try {
    const response = await api('/api/ecosystem/models', { body });
    const updated = withSavedAssignments(block, response);
    blocks.set(block.id, updated);
    // Se repinta la tabla con lo guardado y se deja el mensaje con los ficheros regenerados.
    const holder = document.createElement('div');
    holder.innerHTML = sectionHtml(updated);
    const fresh = holder.firstElementChild;
    section.replaceWith(fresh);
    showStatus(fresh, [savedMessage(response), ...(response.notes || [])].join(' '), 'ok');
    toast(`Modelos de ${block.name} guardados`);
  } catch (err) {
    showErrors(section, violationsFromError(err));
    showStatus(section, '');
    toast(`No se pudieron guardar los modelos de ${block.name}`, 'bad');
  } finally {
    restore();
  }
}

// bindModelsEditor cablea los eventos una sola vez sobre el contenedor: el HTML interior se
// repinta en cada carga.
export function bindModelsEditor(root) {
  root.addEventListener('change', event => {
    const select = event.target.closest('[data-pm-model]');
    const tr = select?.closest('tr[data-phase]');
    const section = select?.closest('[data-adapter]');
    if (tr && section) onModelChange(section, tr);
  });
  root.addEventListener('click', event => {
    const button = event.target.closest('[data-pm-save]');
    const section = button?.closest('[data-adapter]');
    if (button && section && !button.disabled) save(section, button);
  });
}
