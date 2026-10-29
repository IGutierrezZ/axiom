// Incrementos: tablero por fase y vista lista, filtros con contador, filtro por rol
// y modal «Nuevo incremento». Los datos vienen de GET /api/increments; los roles del
// filtro, de /api/workspace (state.workspace.roles).

import { api, escapeHtml } from '../core/api.js';
import { $, $all, openDialog, closeDialog, setBusy, chip, button, stateEmpty, stateError, tableWrap } from '../core/components.js';
import { icon } from '../core/icons.js';
import { PHASES, phaseIndex, typeInfo, typeChip, flagBadge, progressBar, miniStepper, formatDate, isArchived } from '../core/increment.js';
import { hrefFor, navigate } from '../core/router.js';
import { state, set } from '../core/state.js';

// Cuántos archivados se muestran, atenuados, en la columna Archive del tablero.
const RECENT_ARCHIVED = 3;
// Una columna vacía se pliega a una franja, salvo las de fases con acción.
const ALWAYS_OPEN = new Set(['verify']);

const FILTERS = [
  { id: 'all', label: 'Activos', test: () => true },
  { id: 'bug', label: 'Bugs', test: inc => typeInfo(inc).id === 'bug' },
  { id: 'spec', label: 'Pendiente spec', test: inc => inc.pending_spec },
  { id: 'design', label: 'Listo para design', test: inc => inc.ready_for_design },
  { id: 'waiting', label: 'Esperando roles', test: inc => inc.waiting_roles },
  { id: 'gverify', label: 'Verificación global', test: inc => inc.ready_for_global_verify },
  { id: 'archive', label: 'Listo para archivar', test: inc => inc.ready_for_archive },
  { id: 'archived', label: 'Archivados', test: isArchived }
];

// ---- Preferencias del visor (solo comodidad; sin almacenamiento no pasa nada) ---------

const PREF_KEYS = { view: 'axiom.inc.view', filter: 'axiom.inc.filter', role: 'axiom.inc.role' };

function readPref(key, fallback, allowed) {
  try {
    const value = window.localStorage.getItem(key);
    if (value !== null && (!allowed || allowed.includes(value))) return value;
  } catch (_) {
    // Sin almacenamiento: se usa el valor por defecto.
  }
  return fallback;
}

function writePref(key, value) {
  try {
    window.localStorage.setItem(key, value);
  } catch (_) {
    // Nada que hacer: la preferencia solo dura esta sesión.
  }
}

const prefs = {
  view: readPref(PREF_KEYS.view, 'board', ['board', 'list']),
  filter: readPref(PREF_KEYS.filter, 'all', FILTERS.map(f => f.id)),
  role: readPref(PREF_KEYS.role, '')
};

let loadState = 'loading'; // loading | ready | error
let loadError = '';

function setPref(name, value) {
  prefs[name] = value;
  writePref(PREF_KEYS[name], value);
}

// ---- Datos derivados --------------------------------------------------------------------

function workspaceRoles() {
  return state.workspace && state.workspace.roles ? Object.keys(state.workspace.roles) : [];
}

// effectiveRole ignora un rol recordado que ya no existe en el workspace.
function effectiveRole() {
  const found = workspaceRoles().find(r => r.toLowerCase() === prefs.role.toLowerCase());
  return found ? found.toLowerCase() : '';
}

function activeFilter() {
  return FILTERS.find(f => f.id === prefs.filter) || FILTERS[0];
}

function matches(filter, inc) {
  return filter.id === 'archived' ? isArchived(inc) : !isArchived(inc) && filter.test(inc);
}

function visibleItems() {
  const role = effectiveRole();
  let items = state.increments.filter(inc => matches(activeFilter(), inc));
  if (role && prefs.filter !== 'archived') {
    items = items.filter(inc => (inc.pending_roles || []).some(r => r.toLowerCase() === role));
  }
  return items;
}

// ---- Controles: vista, filtros y roles -------------------------------------------------------

function renderControls() {
  $all('#inc-view-toggle [data-view]').forEach(btn => {
    btn.setAttribute('aria-pressed', String(btn.getAttribute('data-view') === prefs.view));
  });

  $('#inc-filters').innerHTML = FILTERS.map(f => chip({
    label: f.label,
    pressed: prefs.filter === f.id,
    count: state.increments.filter(inc => matches(f, inc)).length,
    attrs: `data-filter="${f.id}"`
  })).join('');

  const roles = workspaceRoles();
  const roleBox = $('#inc-role-filters');
  roleBox.hidden = roles.length === 0;
  const role = effectiveRole();
  const disabled = prefs.filter === 'archived' ? ' disabled' : '';
  roleBox.innerHTML = roles.length === 0 ? '' :
    `<span class="ix-label">Rol con tareas pendientes</span>${roles.map(r =>
      `<button type="button" class="chip mono" aria-pressed="${role === r.toLowerCase()}" data-role="${escapeHtml(r)}"${disabled}>${escapeHtml(r)}</button>`
    ).join('')}`;
}

// ---- Tarjeta (tablero) y fila (lista) ----------------------------------------------------------

function dateHtml(inc) {
  const text = formatDate(inc.date);
  return text ? `<time class="ix-date" datetime="${escapeHtml(inc.date)}">${escapeHtml(text)}</time>` : '';
}

function cardHtml(inc) {
  const archived = isArchived(inc);
  const link = hrefFor('incremento', { change: inc.name });
  const progress = inc.tasks_total > 0
    ? `<div class="ix-progress"><div class="ix-row"><span>${inc.tasks_completed}/${inc.tasks_total} tareas</span><span class="mono">${inc.progress_pct}%</span></div>${progressBar(inc.progress_pct, `Tareas de ${inc.name}`)}</div>`
    : (archived ? '' : '<span class="ix-note">Sin tareas todavía</span>');
  const flag = archived ? `<span class="badge badge-neutral">${icon('archive', 13)}Archivado</span>` : flagBadge(inc);
  const footer = [flag, dateHtml(inc)].filter(Boolean).join('');
  return `<article class="ix-card${archived ? ' is-archived' : ''}">
    <div class="ix-card-top">${typeChip(inc)}</div>
    <h4 class="ix-name"><a href="${link}">${escapeHtml(inc.name)}</a></h4>
    ${progress}
    ${footer ? `<div class="ix-card-foot">${footer}</div>` : ''}
  </article>`;
}

function boardHtml(items, archivedExtra) {
  const byPhase = PHASES.map(() => []);
  items.concat(archivedExtra).forEach(inc => byPhase[phaseIndex(inc.phase)].push(inc));
  const cols = PHASES.map((phase, i) => {
    const list = byPhase[i];
    const collapsed = list.length === 0 && !ALWAYS_OPEN.has(phase.id);
    const head = `<div class="ix-col-head"><span class="ix-col-num">0${i + 1}</span><h3 class="ix-col-title" id="ix-h-${phase.id}">${escapeHtml(phase.label)}</h3><span class="ix-col-count">${list.length}<span class="sr-only"> ${list.length === 1 ? 'incremento' : 'incrementos'}</span></span></div>`;
    const body = collapsed ? '' :
      (list.length === 0
        ? '<p class="ix-col-empty">Sin incrementos en esta fase</p>'
        : `<ul class="ix-col-list">${list.map(inc => `<li>${cardHtml(inc)}</li>`).join('')}</ul>`);
    return { collapsed, html: `<section class="ix-col${collapsed ? ' is-collapsed' : ''}" aria-labelledby="ix-h-${phase.id}">${head}${body}</section>` };
  });
  const template = cols.map(c => (c.collapsed ? '3.25rem' : 'minmax(9.25rem, 1fr)')).join(' ');
  return `<div class="ix-board-wrap" role="region" aria-label="Tablero de incrementos por fase" tabindex="0"><div class="ix-board" style="--ix-cols:${template}">${cols.map(c => c.html).join('')}</div></div>`;
}

function listHtml(items) {
  const rows = items.map(inc => {
    const progress = inc.tasks_total > 0 ? `${inc.tasks_completed}/${inc.tasks_total} · ${inc.progress_pct}%` : 'Sin tareas';
    return `<tr>
      <td><a class="ix-name-link" href="${hrefFor('incremento', { change: inc.name })}">${escapeHtml(inc.name)}</a></td>
      <td>${typeChip(inc)}</td>
      <td><span class="ix-phase-cell">${miniStepper(inc)}</span></td>
      <td class="mono">${escapeHtml(progress)}</td>
      <td>${isArchived(inc) ? `<span class="badge badge-neutral">${icon('archive', 13)}Archivado</span>` : flagBadge(inc)}${dateHtml(inc)}</td>
    </tr>`;
  }).join('');
  return tableWrap(`<table class="data-table ix-table"><thead><tr><th scope="col">Incremento</th><th scope="col">Tipo</th><th scope="col">Fase</th><th scope="col">Progreso</th><th scope="col">Estado</th></tr></thead><tbody>${rows}</tbody></table>`, 'Lista de incrementos');
}

// ---- Pintado ------------------------------------------------------------------------------------

// focusKey identifica el control enfocado dentro de la vista para recuperarlo
// después de repintar (los chips se regeneran y, si no, el teclado perdería el foco).
function focusKey() {
  const el = document.activeElement;
  const view = $('#view-incrementos');
  if (!el || !view || !view.contains(el)) return null;
  for (const attr of ['data-filter', 'data-role', 'data-action']) {
    if (el.hasAttribute(attr)) return { attr, value: el.getAttribute(attr) };
  }
  return null;
}

function restoreFocus(key) {
  if (!key) return;
  const view = $('#view-incrementos');
  let target = $all(`[${key.attr}]`, view).find(el => el.getAttribute(key.attr) === key.value && !el.disabled);
  if (!target && key.attr === 'data-action') target = $('.ix-more button', view);
  if (target) target.focus();
}

export function renderIncrements() {
  const key = focusKey();
  paint();
  restoreFocus(key);
}

function paint() {
  const container = $('#increments-container');
  if (!container) return;
  renderControls();

  if (loadState === 'loading') return;
  if (loadState === 'error') {
    container.innerHTML = `${stateError(loadError, 'No se pudieron cargar los incrementos')}<p class="ix-more">${button({ label: 'Reintentar', iconName: 'refresh', size: 'sm', attrs: 'data-action="retry"' })}</p>`;
    announce('No se pudieron cargar los incrementos');
    return;
  }
  if (state.increments.length === 0) {
    container.innerHTML = stateEmpty('Aún no hay incrementos', 'Crea el primero con «Nuevo incremento» en la barra superior.', 'board');
    announce('No hay incrementos');
    return;
  }

  const items = visibleItems();
  const archivedAll = state.increments.filter(isArchived);
  const showArchived = prefs.filter === 'all' && !effectiveRole() && prefs.view === 'board';
  const archivedExtra = showArchived ? archivedAll.slice(0, RECENT_ARCHIVED) : [];

  let body;
  if (items.length === 0) {
    body = `<div class="state">${icon('inbox', 24)}<span class="state-title">Ningún incremento coincide con este filtro</span>${button({ label: 'Quitar filtros', size: 'sm', attrs: 'data-action="reset"' })}</div>`;
  } else {
    body = prefs.view === 'list' ? listHtml(items) : boardHtml(items, archivedExtra);
  }

  let more = '';
  if (prefs.filter === 'archived') {
    more = button({ label: 'Volver a los activos', iconName: 'arrowUp', size: 'sm', variant: 'ghost', attrs: 'data-action="active"' });
  } else if (archivedAll.length > 0) {
    more = button({ label: `Ver los ${archivedAll.length} archivados`, iconName: 'archive', size: 'sm', variant: 'ghost', attrs: 'data-action="archived"' });
  }
  container.innerHTML = body + (more ? `<p class="ix-more">${more}</p>` : '');

  const n = items.length;
  announce(`${n} ${n === 1 ? 'incremento' : 'incrementos'} en ${activeFilter().label}`);
}

function announce(text) {
  const live = $('#inc-live');
  if (live) live.textContent = text;
}

export async function loadIncrements() {
  try {
    const data = await api('/api/increments');
    set('increments', data || []);
    loadState = 'ready';
  } catch (err) {
    console.error(err);
    loadState = 'error';
    loadError = err.message;
  }
  renderIncrements();
}

// ---- Eventos ------------------------------------------------------------------------------------

function initControls() {
  $('#inc-view-toggle').addEventListener('click', (event) => {
    const btn = event.target.closest('[data-view]');
    if (!btn) return;
    setPref('view', btn.getAttribute('data-view'));
    renderIncrements();
  });

  $('#inc-filters').addEventListener('click', (event) => {
    const btn = event.target.closest('[data-filter]');
    if (!btn) return;
    setPref('filter', btn.getAttribute('data-filter'));
    renderIncrements();
  });

  $('#inc-role-filters').addEventListener('click', (event) => {
    const btn = event.target.closest('[data-role]');
    if (!btn || btn.disabled) return;
    const role = btn.getAttribute('data-role');
    setPref('role', effectiveRole() === role.toLowerCase() ? '' : role);
    renderIncrements();
  });

  $('#increments-container').addEventListener('click', (event) => {
    const btn = event.target.closest('[data-action]');
    if (!btn) return;
    switch (btn.getAttribute('data-action')) {
      case 'retry':
        loadState = 'loading';
        renderIncrements();
        loadIncrements();
        return;
      case 'reset':
        setPref('filter', 'all');
        setPref('role', '');
        break;
      case 'archived':
        setPref('filter', 'archived');
        break;
      case 'active':
        setPref('filter', 'all');
        break;
      default:
        return;
    }
    renderIncrements();
  });
}

// ---- Modal «Nuevo incremento» ------------------------------------------------------------------

function initNewIncrement() {
  const dialog = $('#modal-new-increment');
  const name = $('#input-inc-name');
  const type = $('#select-inc-type');
  const intent = $('#input-inc-intent');
  const errorBox = $('#new-inc-error');
  const submit = $('#btn-submit-new-increment');

  const clearInvalid = () => {
    name.removeAttribute('aria-invalid');
    intent.removeAttribute('aria-invalid');
  };
  const fail = (message, field) => {
    clearInvalid();
    errorBox.hidden = false;
    errorBox.textContent = message;
    if (field) {
      field.setAttribute('aria-invalid', 'true');
      field.focus();
    }
  };

  $('#btn-new-increment').addEventListener('click', () => {
    name.value = '';
    intent.value = '';
    type.value = 'feature';
    errorBox.hidden = true;
    clearInvalid();
    openDialog(dialog, '#input-inc-name');
  });

  submit.addEventListener('click', async () => {
    const nameVal = name.value.trim();
    const intentVal = intent.value.trim();
    if (!nameVal || !/^[a-z0-9]+(-[a-z0-9]+)*$/.test(nameVal)) {
      fail('El nombre del incremento debe seguir el formato kebab-case (ej. inc-16-metricas-rendimiento).', name);
      return;
    }
    if (!intentVal) {
      fail('Debes indicar la intención o el propósito técnico del incremento.', intent);
      return;
    }
    errorBox.hidden = true;
    clearInvalid();
    const restore = setBusy(submit, 'Creando…');
    try {
      const data = await api('/api/increments', { body: { name: nameVal, type: type.value, intent: intentVal } });
      if (data && data.success === false) throw new Error(data.error || 'Error al crear el incremento');
      closeDialog(dialog);
      await loadIncrements();
      navigate('incremento', { change: nameVal });
    } catch (err) {
      fail(`Error: ${err.message}`);
    } finally {
      restore();
    }
  });
}

export function initIncrements() {
  initControls();
  initNewIncrement();
  renderControls();
}
