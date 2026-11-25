// Detalle de incremento: cabecera (con la orden de agente copiable), pasos de fase
// y pestañas Artefactos · Roles y barrera · Handoff, sincronizadas con la ruta por
// hash. Roles y Handoff viven en detalle-roles.js y detalle-handoff.js.

import { api, escapeHtml } from '../core/api.js';
import { $, $all, button, commandChip, stateEmpty, stateError, stateLoading, toast } from '../core/components.js';
import { icon } from '../core/icons.js';
import { detailSteps, formatDate, isArchived, nextCommand, progressBar, typeChip } from '../core/increment.js';
import { firstParagraph, renderMarkdown } from '../core/markdown.js';
import { hrefFor } from '../core/router.js';
import { barrierSummary, renderRoles } from './detalle-roles.js';
import { initHandoff, renderHandoff } from './detalle-handoff.js';

// Estado de la vista abierta. token invalida respuestas tardías al cambiar de ruta.
const ctx = { change: null, failed: null, detail: null, tab: 'artefactos', token: 0, artifact: null, raw: false };

function incrementUrl(change) {
  return `/api/increments/${encodeURIComponent(change)}`;
}

// ---- Cabecera y pasos --------------------------------------------------------------------

function renderHeader(detail) {
  const s = detail.summary;
  const date = formatDate(s.date);
  $('#inc-meta').innerHTML = [
    typeChip(s),
    isArchived(s) ? `<span class="badge badge-neutral">${icon('archive', 13)}Archivado</span>` : '',
    `<span class="badge badge-mono">Fase: ${escapeHtml(s.phase)}</span>`,
    s.tasks_total > 0 ? `<span class="badge badge-neutral">Tareas: ${s.tasks_completed}/${s.tasks_total} (${s.progress_pct}%)</span>` : '',
    date ? `<time class="badge badge-neutral" datetime="${escapeHtml(s.date)}">${escapeHtml(date)}</time>` : '',
    commandChip(nextCommand(s))
  ].filter(Boolean).join('');
  $('#inc-title').textContent = s.name;
  const desc = firstParagraph(detail.proposal);
  const descEl = $('#inc-desc');
  descEl.hidden = !desc;
  descEl.textContent = desc;
}

const ROLE_TONE = { ok: 'text-ok' };

// stepRoles: en multirol, una fila por rol dentro del paso Apply y el enlace a la
// pestaña «Roles y barrera»; con la barrera cerrada, también su primer bloqueo.
function stepRoles(st, change) {
  if (!st.roles) return '';
  const rows = st.roles.map(r => {
    const tone = ROLE_TONE[r.status.tone] || '';
    return `<li class="dt-step-role"><span class="dt-step-role-name" title="${escapeHtml(r.label)}">${escapeHtml(r.label)}</span><span class="dt-step-role-state mono${tone ? ` ${tone}` : ''}">${escapeHtml(r.status.text)}</span>${progressBar(r.pct, `Tareas del rol ${r.label}`)}</li>`;
  }).join('');
  const blocker = st.blocker ? `<span class="dt-step-role-blocker" title="${escapeHtml(st.blocker)}">${escapeHtml(st.blocker)}</span>` : '';
  const link = `<a class="dt-step-role-link" href="${hrefFor('incremento', { change, tab: 'roles' })}">${icon(st.waiting ? 'clock' : 'users', 12)}${st.waiting ? 'Barrera cerrada · ver roles' : 'Ver roles'}</a>`;
  return `<ul class="dt-step-roles" aria-label="Avance por rol">${rows}</ul>${blocker}${link}`;
}

function renderStepper(detail) {
  const steps = detailSteps(detail);
  const change = detail.summary.name;
  const stateText = { done: 'completada', current: 'en curso', todo: 'pendiente' };
  $('#inc-stepper').innerHTML = `<ol class="dt-steps" aria-label="Fases SDD">${steps.map(st => {
    const dot = st.state === 'done' ? icon('check', 12) : st.state === 'current' ? '<span class="dt-step-core"></span>' : '';
    const sub = st.detail ? `${escapeHtml(st.detail)}${st.pct != null ? ` · ${st.pct}%` : ''}` : escapeHtml(st.file);
    const waiting = st.waiting ? ', barrera cerrada' : '';
    return `<li class="dt-step is-${st.state}${st.waiting ? ' is-waiting' : ''}"${st.state === 'current' ? ' aria-current="step"' : ''}>
      <span class="dt-step-track"><span class="dt-step-dot">${dot}</span><span class="dt-step-bar"></span></span>
      <span class="dt-step-label">${escapeHtml(st.label)}<span class="sr-only"> — ${stateText[st.state]}${waiting}</span></span>
      <span class="dt-step-sub mono">${sub}</span>
      ${stepRoles(st, change)}
    </li>`;
  }).join('')}</ol>`;
}

// ---- Pestañas --------------------------------------------------------------------------------------

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

function renderTabPill(detail) {
  const pill = $('#inc-tab-roles-pill');
  const sum = barrierSummary(detail.barrier_report);
  pill.hidden = !sum;
  if (sum) {
    pill.className = `tab-pill tab-pill-${sum.tone}`;
    pill.textContent = sum.text;
  }
}

// ---- Artefactos ----------------------------------------------------------------------------------------

// Construye la lista de documentos desde el detalle, en este orden: proposal, cada spec de dominio,
// design, cada fichero de tareas, verify y archive. Cada documento lleva un id estable. Si no hay
// specs o tareas se deja un único marcador ausente con el nombre habitual del fichero.
function availableArtifacts(detail) {
  const single = (id, flag, body, file) => ({ id, file, exists: !!detail[flag], text: detail[body] || '' });
  const files = (kind, list, flag, placeholder) => {
    const items = Array.isArray(list) ? list : [];
    if (items.length === 0) return [{ id: kind, kind, file: placeholder, exists: false, text: '' }];
    return items.map(f => ({ id: `${kind}:${f.path}`, kind, file: f.path, exists: true, text: f.content || '' }));
  };
  return [
    single('proposal', 'has_proposal', 'proposal', 'proposal.md'),
    ...files('spec', detail.specs, 'has_spec', 'specs/<dominio>/spec.md'),
    single('design', 'has_design', 'design', 'design.md'),
    ...files('tasks', detail.tasks, 'has_tasks', 'tasks.md'),
    single('verify', 'has_verify', 'verify_report', 'verify-report.md'),
    single('archive', 'has_archive', 'archive_report', 'archive-report.md')
  ];
}

// El progreso agregado del cambio solo se atribuye a las tareas cuando hay un único fichero.
function artifactMeta(a, detail) {
  if (a.kind === 'tasks' && a.exists && (detail.tasks || []).length === 1 && detail.summary.tasks_total > 0) {
    return `${detail.summary.tasks_completed}/${detail.summary.tasks_total}`;
  }
  return '';
}

function renderArtifacts() {
  const detail = ctx.detail;
  const panel = $('#inc-panel-artefactos');
  const list = availableArtifacts(detail);
  const existing = list.filter(a => a.exists);
  if (existing.length === 0) {
    panel.innerHTML = stateEmpty('Este incremento aún no tiene artefactos', 'Aparecerán a medida que avancen las fases SDD.', 'file');
    return;
  }
  if (!existing.some(a => a.id === ctx.artifact)) ctx.artifact = existing[0].id;
  const selected = existing.find(a => a.id === ctx.artifact);

  const nav = list.map(a => {
    if (!a.exists) {
      return `<li><span class="dt-doc is-missing">${icon('fileDashed', 16)}<span class="mono grow">${escapeHtml(a.file)}</span><span class="dt-doc-meta">no existe todavía</span></span></li>`;
    }
    const meta = artifactMeta(a, detail);
    const current = a.id === ctx.artifact;
    return `<li><button type="button" class="dt-doc" data-artifact="${escapeHtml(a.id)}"${current ? ' aria-current="true"' : ''}>${icon('file', 16)}<span class="mono grow">${escapeHtml(a.file)}</span>${meta ? `<span class="dt-doc-meta mono">${escapeHtml(meta)}</span>` : ''}</button></li>`;
  }).join('');

  let body;
  if (!selected.text.trim()) {
    body = stateEmpty('No hay contenido que mostrar', 'El artefacto existe, pero el servidor no ha devuelto su texto (puede estar repartido en varios ficheros).', 'file');
  } else if (ctx.raw) {
    body = `<pre class="md-raw" tabindex="0">${escapeHtml(selected.text)}</pre>`;
  } else {
    body = `<div class="md">${renderMarkdown(selected.text, { sectionStats: selected.kind === 'tasks' })}</div>`;
  }

  panel.innerHTML = `<div class="dt-artifacts">
    <nav class="dt-doclist" aria-label="Artefactos del incremento"><ul>${nav}</ul></nav>
    <article class="dt-viewer" aria-labelledby="dt-viewer-title">
      <div class="dt-viewer-head">
        <h2 id="dt-viewer-title" class="mono">${escapeHtml(selected.file)}</h2>
        <div class="seg-group seg-inline" role="group" aria-label="Formato del documento">
          <button type="button" class="seg seg-wide" data-format="view" aria-pressed="${!ctx.raw}">Vista</button>
          <button type="button" class="seg seg-wide" data-format="raw" aria-pressed="${ctx.raw}">Markdown</button>
        </div>
      </div>
      <div class="dt-viewer-body">${body}</div>
    </article>
  </div>`;
}

// withFocus repinta conservando el foco del teclado en el control equivalente.
function withFocus(fn) {
  const el = document.activeElement;
  let sel = null;
  if (el && $('#inc-panel-artefactos').contains(el)) {
    if (el.hasAttribute('data-artifact')) sel = `[data-artifact="${CSS.escape(el.getAttribute('data-artifact'))}"]`;
    else if (el.hasAttribute('data-format')) sel = `[data-format="${el.getAttribute('data-format')}"]`;
  }
  fn();
  if (sel) {
    const target = $(sel, $('#inc-panel-artefactos'));
    if (target) target.focus();
  }
}

// ---- Montaje ------------------------------------------------------------------------------------------------

function paintDetail() {
  const detail = ctx.detail;
  renderHeader(detail);
  renderStepper(detail);
  renderTabPill(detail);
  renderArtifacts();
}

async function renderTabContent() {
  const { change, tab, detail } = ctx;
  if (!detail) return;
  const token = ctx.token;
  const isCurrent = () => token === ctx.token && ctx.change === change;
  if (tab === 'roles') {
    await renderRoles($('#inc-panel-roles'), { detail, change, onMigrated: reloadDetail });
  } else if (tab === 'handoff') {
    await renderHandoff($('#inc-panel-handoff'), change, { archived: isArchived(detail.summary), isCurrent });
  }
}

// refreshDetail repinta el detalle abierto con datos nuevos (botón Actualizar).
export function refreshDetail() {
  return reloadDetail();
}

// reloadDetail vuelve a pedir el detalle sin vaciar la pantalla (tras Migrar o un handoff).
async function reloadDetail() {
  const change = ctx.change;
  if (!change) return;
  const token = ++ctx.token;
  try {
    const detail = await api(incrementUrl(change));
    if (token !== ctx.token) return;
    ctx.detail = detail;
    paintDetail();
    await renderTabContent();
  } catch (err) {
    if (token === ctx.token) toast(`No se pudo actualizar el incremento: ${err.message}`, 'bad');
  }
}

function showLoading(change) {
  $('#inc-title').textContent = change;
  $('#inc-meta').innerHTML = '';
  $('#inc-desc').hidden = true;
  $('#inc-stepper').innerHTML = stateLoading('Cargando el incremento…');
  $('#inc-tab-roles-pill').hidden = true;
  $('#inc-panel-artefactos').innerHTML = stateLoading('Cargando artefactos…');
}

function showLoadError(err) {
  $('#inc-stepper').innerHTML = '';
  $('#inc-panel-artefactos').innerHTML = `${stateError(err.message, 'No se pudo obtener el incremento')}<p class="ix-more">${button({ label: 'Reintentar', iconName: 'refresh', size: 'sm', attrs: 'data-action="retry"' })}</p>`;
}

// enterDetail monta la vista del incremento de la ruta (carga el detalle si cambió).
export async function enterDetail(route) {
  const { change, tab } = route;
  ctx.tab = tab;
  renderTabs(change, tab);

  if (ctx.change !== change || !ctx.detail) {
    if (ctx.change !== change) {
      // La selección de artefacto y el formato solo se reinician al cambiar de incremento.
      ctx.artifact = null;
      ctx.raw = false;
    }
    ctx.change = change;
    ctx.detail = null;
    const token = ++ctx.token;
    showLoading(change);
    try {
      const detail = await api(incrementUrl(change));
      if (token !== ctx.token) return;
      ctx.detail = detail;
      paintDetail();
    } catch (err) {
      if (token !== ctx.token) return;
      showLoadError(err);
      ctx.failed = change;
      ctx.change = null;
      return;
    }
  }
  await renderTabContent();
}

export function initDetail() {
  $('#inc-panel-artefactos').addEventListener('click', (event) => {
    const art = event.target.closest('[data-artifact]');
    const fmt = event.target.closest('[data-format]');
    if (art) {
      ctx.artifact = art.getAttribute('data-artifact');
      withFocus(renderArtifacts);
    } else if (fmt) {
      ctx.raw = fmt.getAttribute('data-format') === 'raw';
      withFocus(renderArtifacts);
    } else if (event.target.closest('[data-action="retry"]') && ctx.failed) {
      enterDetail({ name: 'incremento', change: ctx.failed, tab: ctx.tab });
    }
  });

  initHandoff({
    panel: $('#inc-panel-handoff'),
    getChange: () => ctx.change,
    onCreated: (change) => {
      if (change === ctx.change) reloadDetail();
      else window.location.hash = hrefFor('incremento', { change, tab: 'handoff' });
    }
  });
  // Los clics en las pestañas son enlaces de ruta (hash): no requieren JS.
}
