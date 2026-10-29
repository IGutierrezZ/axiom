// Incrementos: tablero provisional (migra en T3) y modal «Nuevo incremento».

import { api, escapeHtml } from '../core/api.js';
import { $, openDialog, closeDialog, setBusy } from '../core/components.js';
import { icon } from '../core/icons.js';
import { hrefFor, navigate } from '../core/router.js';
import { state, set } from '../core/state.js';

let currentFilter = 'all';

// isFix replica la regla histórica: bandera is_bug, tipo fix o nombre con fix/bug/hotfix.
function isFix(inc) {
  const name = inc.name.toLowerCase();
  return inc.is_bug || inc.change_type === 'fix' || name.includes('fix') || name.includes('bug') || name.includes('hotfix');
}

function matchesFilter(inc) {
  switch (currentFilter) {
    case 'all':
    case 'active':
      return inc.type === 'active';
    case 'archived':
      return inc.type === 'archived';
    case 'type:fix':
      return inc.type === 'active' && isFix(inc);
    case 'spec_pending':
      return inc.pending_spec;
    case 'ready_for_design':
      return inc.ready_for_design;
    case 'ready_for_global_verify':
      return inc.ready_for_global_verify;
    case 'ready_for_archive':
      return inc.ready_for_archive;
    default:
      if (currentFilter.startsWith('role:')) {
        const role = currentFilter.split(':')[1].toLowerCase();
        return (inc.pending_roles || []).some(r => r.toLowerCase() === role);
      }
      return true;
  }
}

function operationalBadge(inc) {
  if (inc.ready_for_design) return '<span class="badge badge-ready-design" title="Especificación completa, lista para el diseño">Listo para design</span>';
  if (inc.pending_spec) return '<span class="badge badge-pending-spec" title="Pendiente de cerrar la especificación">Pendiente spec</span>';
  if (inc.waiting_roles && inc.pending_roles && inc.pending_roles.length > 0) {
    return `<span class="badge badge-pending-roles" title="Roles técnicos con tareas pendientes">Pendiente: ${escapeHtml(inc.pending_roles.join(', '))}</span>`;
  }
  if (inc.ready_for_global_verify) return '<span class="badge badge-global-verify" title="Todos los roles han terminado; falta la verificación global">Verif. global</span>';
  if (inc.ready_for_archive) return '<span class="badge badge-ready-archive" title="Verificación superada, listo para archivar">Listo para archivar</span>';
  return '';
}

export function renderIncrements() {
  const container = $('#increments-container');
  const filtered = state.increments.filter(matchesFilter);
  if (filtered.length === 0) {
    container.innerHTML = '<div class="state">No se encontraron incrementos para este filtro.</div>';
    return;
  }
  container.innerHTML = filtered.map(inc => {
    const fix = isFix(inc);
    const kind = fix
      ? `<span class="badge badge-bug">${icon('bug', 13)}Bug / fix</span>`
      : `<span class="badge badge-feature">${icon('sparkles', 13)}Feature</span>`;
    const archived = inc.type === 'archived';
    const done = inc.progress_pct >= 100;
    return `<div class="inc-card ${escapeHtml(inc.type)}">
      <div>
        <div class="card-header">
          <div class="card-title"><a href="${hrefFor('incremento', { change: inc.name })}">${escapeHtml(inc.name)}</a></div>
          <div class="card-badges">${kind}<span class="badge badge-${escapeHtml(inc.type)}">${archived ? 'Archivado' : 'Activo'}</span><span class="badge badge-phase">${escapeHtml(inc.phase)}</span>${operationalBadge(inc)}</div>
        </div>
        <div style="margin: 14px 0">
          <div style="display:flex;justify-content:space-between;font-size:12px;color:var(--text-2);margin-bottom:6px"><span>Tareas: ${inc.tasks_completed}/${inc.tasks_total}</span><span>${inc.progress_pct}%</span></div>
          <div class="progress${done ? ' is-done' : ''}" role="img" aria-label="${inc.progress_pct}% de tareas completadas"><span style="width:${inc.progress_pct}%"></span></div>
        </div>
      </div>
      <div class="card-footer">
        <span class="card-date">${inc.date ? 'Fecha: ' + escapeHtml(inc.date) : 'En progreso'}</span>
        <a class="btn btn-sm" href="${hrefFor('incremento', { change: inc.name })}">Ver detalle</a>
      </div>
    </div>`;
  }).join('');
}

export function updateRoleFilterButtons() {
  const container = $('#role-filters-container');
  container.innerHTML = '';
  const roles = state.workspace && state.workspace.roles ? Object.keys(state.workspace.roles) : [];
  roles.forEach(roleId => {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'filter-btn';
    btn.setAttribute('data-filter', `role:${roleId.toLowerCase()}`);
    btn.setAttribute('aria-pressed', String(currentFilter === `role:${roleId.toLowerCase()}`));
    btn.textContent = roleId;
    container.appendChild(btn);
  });
}

export async function loadIncrements() {
  const container = $('#increments-container');
  try {
    const data = await api('/api/increments');
    set('increments', data || []);
    renderIncrements();
  } catch (err) {
    console.error(err);
    container.innerHTML = `<div class="state state-error" role="alert">${icon('alert', 24)}<span class="state-title">No se pudieron cargar los incrementos</span><span>${escapeHtml(err.message)}</span></div>`;
  }
}

function initFilters() {
  // Delegación: también cubre los filtros de rol que se regeneran.
  $('#increments-filter-group').addEventListener('click', (event) => {
    const btn = event.target.closest('.filter-btn');
    if (!btn) return;
    currentFilter = btn.getAttribute('data-filter');
    document.querySelectorAll('#increments-filter-group .filter-btn').forEach(b => {
      b.setAttribute('aria-pressed', String(b === btn));
    });
    renderIncrements();
  });
}

// ---- Modal «Nuevo incremento» ------------------------------------------------

function initNewIncrement() {
  const dialog = $('#modal-new-increment');
  const name = $('#input-inc-name');
  const type = $('#select-inc-type');
  const intent = $('#input-inc-intent');
  const errorBox = $('#new-inc-error');
  const submit = $('#btn-submit-new-increment');

  const fail = (message, field) => {
    errorBox.hidden = false;
    errorBox.textContent = message;
    if (field) field.focus();
  };

  $('#btn-new-increment').addEventListener('click', () => {
    name.value = '';
    intent.value = '';
    type.value = 'feature';
    errorBox.hidden = true;
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
  initFilters();
  initNewIncrement();
}
