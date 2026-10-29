// Detalle de incremento: Artefactos · Roles y barrera · Handoff.
// Provisional hasta T3. Aquí viven, bajo la ruta del incremento, las antiguas
// pestañas globales Roles y Handoffs (el cambio ya no se elige en un desplegable).

import { api, escapeHtml } from '../core/api.js';
import { $, $all, openDialog, closeDialog, renderConsole, setBusy, toast, statusPill, stateEmpty, stateError, stateLoading } from '../core/components.js';
import { icon } from '../core/icons.js';
import { hrefFor, navigate } from '../core/router.js';
import { state } from '../core/state.js';
import { loadIncrements } from './incrementos.js';

let currentChange = null;
let currentDetail = null;

const ARTIFACTS = [
  ['has_proposal', 'Propuesta', 'proposal.md'],
  ['has_spec', 'Especificación', 'spec.md'],
  ['has_design', 'Diseño técnico', 'design.md'],
  ['has_tasks', 'Plan de tareas', 'tasks.md'],
  ['has_verify', 'Informe de verificación', 'verify-report.md'],
  ['has_archive', 'Informe de archivado', 'archive-report.md']
];

function incrementUrl(change) {
  return `/api/increments/${encodeURIComponent(change)}`;
}

// ---- Cabecera, pestañas y paneles -------------------------------------------

function renderTabs(change, tab) {
  $all('#inc-tabs .tab-link').forEach(link => {
    const t = link.getAttribute('data-tab');
    link.setAttribute('href', hrefFor('incremento', { change, tab: t }));
    if (t === tab) link.setAttribute('aria-current', 'page');
    else link.removeAttribute('aria-current');
  });
  $all('#view-incremento [data-panel]').forEach(panel => {
    panel.hidden = panel.getAttribute('data-panel') !== tab;
  });
}

function renderHeader(detail) {
  const s = detail.summary;
  $('#inc-title').textContent = s.name;
  const archived = s.type === 'archived';
  $('#inc-badges').innerHTML = [
    statusPill(archived ? 'ok' : 'accent', archived ? 'Archivado' : 'Activo'),
    `<span class="badge badge-mono">Fase: ${escapeHtml(s.phase)}</span>`,
    `<span class="badge badge-neutral">Tareas: ${s.tasks_completed}/${s.tasks_total} (${s.progress_pct}%)</span>`
  ].join('');
  $('#btn-inc-continue').disabled = archived;
  $('#btn-inc-verify').disabled = archived;
}

function renderArtifacts(detail) {
  const rows = ARTIFACTS.map(([flag, label, file]) => `<div class="row">
      <div class="row-main"><span style="font-weight:600">${escapeHtml(label)}</span><span class="mono dim" style="font-size:12px">${escapeHtml(file)}</span></div>
      ${detail[flag] ? statusPill('ok', 'Presente') : statusPill('bad', 'Ausente')}
    </div>`).join('');

  let preview = '';
  if (detail.proposal) {
    const text = detail.proposal.length > 500 ? detail.proposal.substring(0, 500) + '…' : detail.proposal;
    preview = `<div class="card"><h2 class="card-title-h">Vista previa de la propuesta</h2><pre class="preview-pre">${escapeHtml(text)}</pre></div>`;
  }
  $('#inc-panel-artefactos').innerHTML = `<div class="card card-flush"><div class="card-head"><h2 class="card-title-h">Artefactos detectados</h2></div><div class="row-list">${rows}</div></div>${preview ? `<div style="margin-top:20px">${preview}</div>` : ''}`;
}

// enterDetail monta la vista del incremento de la ruta (carga el detalle si cambió).
export async function enterDetail(route) {
  const { change, tab } = route;
  renderTabs(change, tab);

  if (currentChange !== change || !currentDetail) {
    // La consola solo se limpia al cambiar de incremento, no al refrescar el mismo.
    if (currentChange !== change) $('#inc-console').hidden = true;
    currentChange = change;
    currentDetail = null;
    $('#inc-title').textContent = change;
    $('#inc-badges').innerHTML = '';
    $('#inc-panel-artefactos').innerHTML = stateLoading('Cargando artefactos…');
    try {
      currentDetail = await api(incrementUrl(change));
      if (currentChange !== change) return;
      renderHeader(currentDetail);
      renderArtifacts(currentDetail);
    } catch (err) {
      $('#inc-panel-artefactos').innerHTML = stateError(err.message, 'No se pudo obtener el incremento');
      currentChange = null;
      return;
    }
  }

  if (tab === 'roles') await loadRolesForChange(change);
  if (tab === 'handoff') await loadHandoffForChange(change);
}

// ---- Acciones SDD (Continuar / Validar verificación) ------------------------

async function runAction(path, change) {
  try {
    const data = await api(path, { body: { name: change } });
    return { ok: !!data && data.success !== false, data: data || {}, error: '' };
  } catch (err) {
    return { ok: false, data: err.data || {}, error: err.message };
  }
}

function initActions() {
  const consoleBox = $('#inc-console');

  const cont = $('#btn-inc-continue');
  cont.addEventListener('click', async () => {
    if (!currentChange) return;
    const change = currentChange;
    const restore = setBusy(cont, 'Ejecutando…');
    const { ok, data, error } = await runAction('/api/increments/continue', change);
    if (ok) {
      renderConsole(consoleBox, { title: 'Continuar SDD', state: 'ok', text: `[ACCIÓN SDD: ${data.action || 'continue'}]\n\n${data.output || 'Transición efectuada con éxito.'}` });
    } else {
      renderConsole(consoleBox, { title: 'Continuar SDD', state: 'error', text: `[ERROR]\n\n${data.error || error || 'Fallo en la transición SDD'}\n\n${data.output || ''}` });
    }
    restore();
    await loadIncrements();
    currentDetail = null;
    const route = { name: 'incremento', change, tab: currentTab() };
    await enterDetail(route);
  });

  const verify = $('#btn-inc-verify');
  verify.addEventListener('click', async () => {
    if (!currentChange) return;
    const restore = setBusy(verify, 'Validando…');
    const { ok, data, error } = await runAction('/api/increments/verify', currentChange);
    if (ok) {
      renderConsole(consoleBox, { title: 'Verificación formal', state: 'ok', text: `[VERIFICACIÓN FORMAL: CORRECTA]\n\n${data.output || 'Conformidad de especificaciones aprobada.'}` });
    } else {
      renderConsole(consoleBox, { title: 'Verificación formal', state: 'error', text: `[VERIFICACIÓN FORMAL: INCONSISTENCIA O ERROR]\n\n${data.error || error || 'Fallo en la verificación'}\n\n${data.output || ''}` });
    }
    restore();
  });
}

function currentTab() {
  const active = document.querySelector('#inc-tabs .tab-link[aria-current="page"]');
  return active ? active.getAttribute('data-tab') : 'artefactos';
}

// ---- Roles y barrera ---------------------------------------------------------

async function loadRolesForChange(change) {
  const rolesContainer = $('#roles-container');
  const banner = $('#barrier-result-card');
  rolesContainer.innerHTML = stateLoading('Evaluando roles y barrera…');
  try {
    const barrier = await api(`/api/roles?change=${encodeURIComponent(change)}`);
    banner.classList.remove('hidden');
    const roleList = Array.isArray(barrier.roles) ? barrier.roles : [];
    const single = roleList.length <= 1 && (roleList.length === 0 || roleList[0].role === 'fullstack');

    if (single) {
      banner.className = 'barrier-banner satisfied';
      $('#barrier-icon').innerHTML = icon('sparkles', 20);
      $('#barrier-title').textContent = 'Rol unificado (fullstack): flujo continuo';
      $('#barrier-desc').textContent = 'Axiom opera con el rol canónico unificado «fullstack». No requiere barreras fan-in concurrentes ni esperas entre agentes: el ciclo avanza de forma continua.';
    } else if (barrier.satisfied) {
      banner.className = 'barrier-banner satisfied';
      $('#barrier-icon').innerHTML = icon('check', 20);
      $('#barrier-title').textContent = 'Barrera superada';
      $('#barrier-desc').textContent = 'Todos los roles obligatorios (blocking) han verificado sus tareas.';
    } else {
      banner.className = 'barrier-banner blocked';
      $('#barrier-icon').innerHTML = icon('x', 20);
      $('#barrier-title').textContent = 'Barrera bloqueada';
      $('#barrier-desc').textContent = (barrier.blockers && barrier.blockers.length > 0)
        ? barrier.blockers.join(' • ')
        : 'Existen roles obligatorios con tareas pendientes o con la verificación fallida.';
    }

    if (roleList.length === 0) {
      rolesContainer.innerHTML = stateEmpty('Sin roles específicos', 'Modo mono-rol: no se detectaron roles declarados.');
      return;
    }
    rolesContainer.innerHTML = roleList.map(status => {
      const verified = status.verdict === 'pass';
      const tasks = status.tasks || { total: 0, completed: 0, percent: 0 };
      const pct = Math.round(tasks.percent);
      return `<div class="role-card">
        <div>
          <div class="card-header">
            <div class="card-title">${escapeHtml(status.name || status.role)}</div>
            <div class="card-badges">
              <span class="badge badge-${escapeHtml(status.gate_policy)}">${escapeHtml(status.gate_policy)}</span>
              ${verified ? statusPill('ok', `Verif.: ${status.verdict}`) : statusPill('bad', `Verif.: ${status.verdict || 'pendiente'}`)}
            </div>
          </div>
          <div style="margin: 14px 0">
            <div style="display:flex;justify-content:space-between;font-size:12px;color:var(--text-2);margin-bottom:6px"><span>Tareas: ${tasks.completed}/${tasks.total}</span><span>${pct}%</span></div>
            <div class="progress${pct >= 100 ? ' is-done' : ''}" role="img" aria-label="${pct}% de tareas completadas"><span style="width:${pct}%"></span></div>
          </div>
          <p class="dim" style="font-size:12px">Repositorios: <code>${escapeHtml((status.repositories || []).join(', ') || '.')}</code></p>
        </div>
      </div>`;
    }).join('');
  } catch (err) {
    console.error(err);
    rolesContainer.innerHTML = stateError(err.message, 'No se pudieron evaluar los roles');
    banner.classList.add('hidden');
  }
}

// ---- Handoff -----------------------------------------------------------------

function showHandoffEmpty(text) {
  $('#handoff-display').classList.add('hidden');
  const empty = $('#handoff-empty');
  empty.classList.remove('hidden');
  empty.textContent = text;
}

async function loadHandoffForChange(change) {
  try {
    const ho = await api(`/api/handoffs?change=${encodeURIComponent(change)}`);
    $('#handoff-empty').classList.add('hidden');
    $('#handoff-display').classList.remove('hidden');
    const meta = ho.metadata || {};
    const sections = ho.sections || {};

    $('#ho-from-phase').textContent = meta.from_phase || 'origen';
    $('#ho-to-phase').textContent = meta.to_phase || 'destino';
    $('#ho-roles').innerHTML = `${escapeHtml(meta.from_role)} ${icon('arrowRight', 12)} ${escapeHtml(meta.to_role)}`;
    const ready = meta.status === 'ready';
    const status = $('#ho-status');
    status.className = `badge ${ready ? 'badge-ok' : 'badge-warn'}`;
    status.innerHTML = `${icon(ready ? 'check' : 'alert', 13)}${escapeHtml(meta.status || 'ready')}`;
    $('#ho-time').textContent = meta.timestamp ? new Date(meta.timestamp).toLocaleString('es-ES') : '--';

    const parts = [
      ['1', 'Resumen ejecutivo', sections.executive_summary],
      ['2', 'Artefactos modificados y creados', sections.artifacts],
      ['3', 'Decisiones técnicas y acuerdos', sections.decisions],
      ['4', 'Riesgos, bloqueos y preguntas abiertas', sections.risks_and_blockers],
      ['5', 'Instrucciones directas para el siguiente rol', sections.direct_instructions]
    ];
    $('#ho-sections-container').innerHTML = parts.map(([n, title, text]) =>
      `<div class="section-box"><h3>${n}. ${escapeHtml(title)}</h3><div class="section-content">${escapeHtml(text || 'Sin contenido.')}</div></div>`
    ).join('');
  } catch (err) {
    if (err.status === 404) showHandoffEmpty(`No se encontró un handoff.md para «${change}».`);
    else showHandoffEmpty(`Error cargando el handoff: ${err.message}`);
  }
}

// ---- Modal «Redactar handoff» -------------------------------------------------

function fillHandoffChangeOptions(selected) {
  const select = $('#handoff-modal-change');
  select.innerHTML = state.increments
    .filter(inc => inc.type === 'active')
    .map(inc => `<option value="${escapeHtml(inc.name)}">${escapeHtml(inc.name)} (fase: ${escapeHtml(inc.phase)})</option>`)
    .join('');
  if (selected) select.value = selected;
}

function initHandoffModal() {
  const dialog = $('#modal-create-handoff');
  const errorBox = $('#handoff-error');
  const submit = $('#btn-submit-create-handoff');
  const val = (id) => $(id).value.trim();

  $('#btn-open-create-handoff').addEventListener('click', () => {
    fillHandoffChangeOptions(currentChange);
    errorBox.hidden = true;
    openDialog(dialog, '#handoff-modal-summary');
  });

  submit.addEventListener('click', async () => {
    const change = $('#handoff-modal-change').value;
    const summary = val('#handoff-modal-summary');
    if (!change) {
      errorBox.hidden = false;
      errorBox.textContent = 'Debes seleccionar un incremento para el handoff.';
      return;
    }
    if (!summary) {
      errorBox.hidden = false;
      errorBox.textContent = 'Debes indicar el resumen ejecutivo del relevo formal.';
      $('#handoff-modal-summary').focus();
      return;
    }
    errorBox.hidden = true;
    const restore = setBusy(submit, 'Registrando…');
    try {
      const data = await api('/api/handoffs', {
        body: {
          change,
          from_phase: $('#handoff-modal-from-phase').value,
          to_phase: $('#handoff-modal-to-phase').value,
          from_role: val('#handoff-modal-from-role'),
          to_role: val('#handoff-modal-to-role'),
          status: $('#handoff-modal-status').value,
          executive_summary: summary,
          artifacts: val('#handoff-modal-artifacts'),
          decisions: val('#handoff-modal-decisions'),
          risks: val('#handoff-modal-risks'),
          instructions: val('#handoff-modal-instructions')
        }
      });
      if (data && data.success === false) throw new Error(data.error || 'Error al registrar el handoff');
      closeDialog(dialog);
      toast('Handoff registrado.', 'ok');
      navigate('incremento', { change, tab: 'handoff' });
    } catch (err) {
      errorBox.hidden = false;
      errorBox.textContent = `Error al registrar el handoff: ${err.message}`;
    } finally {
      restore();
    }
  });
}

export function initDetail() {
  initActions();
  initHandoffModal();
  // Los clics en las pestañas son enlaces de ruta (hash): no requieren JS.
}
