// Skills: buzón de propuestas con aprobación humana (lista y vista previa) y catálogo de
// skills activas con filtro. Aprobar y rechazar piden confirmación (D5) y refrescan el
// buzón, la lista activa y la insignia de la barra lateral.

import { api, escapeHtml } from '../core/api.js';
import { $, badge, confirmDialog, setBusy, stateEmpty, stateError, statusPill, toast } from '../core/components.js';
import { formatDateTime, formatNumber, plural, shortHash } from '../core/format.js';
import { icon } from '../core/icons.js';
import { renderMarkdown } from '../core/markdown.js';
import { refreshOverview } from '../core/overview.js';

const ORIGIN_LABEL = { midudev: 'midudev (auditado)', mined: 'Minería local' };

const ctx = { proposals: [], selected: null, raw: false, active: [], filter: '' };

// ---- Estado y avisos en línea ------------------------------------------------------------

// feedback pinta un aviso persistente bajo la cabecera (el toast desaparece solo).
// kind: ok | warn | bad | info.
function feedback(kind, html) {
  const box = $('#scan-feedback');
  if (!box) return;
  const iconName = { ok: 'check', warn: 'alert', bad: 'alert', info: 'info' }[kind] || 'info';
  box.className = `alert alert-${kind === 'info' ? 'neutral' : kind}`;
  box.setAttribute('role', kind === 'bad' ? 'alert' : 'status');
  box.innerHTML = `${icon(iconName, 18)}<div class="grow">${html}</div>`;
}

// ---- Skills activas ------------------------------------------------------------------------

function renderActive() {
  const container = $('#skills-container');
  const count = $('#skills-count');
  const q = ctx.filter.trim().toLowerCase();
  const all = ctx.active;
  const list = q
    ? all.filter(s => [s.name, s.description, s.trigger].some(v => String(v || '').toLowerCase().includes(q)))
    : all;
  count.textContent = q ? `skills/ · ${formatNumber(list.length)} de ${formatNumber(all.length)}` : `skills/ · ${formatNumber(all.length)}`;
  if (all.length === 0) {
    container.innerHTML = stateEmpty('No hay skills activas', 'Aún no hay skills instaladas en skills/. Aprueba alguna propuesta del buzón o usa «Escanear».', 'cube');
    return;
  }
  if (list.length === 0) {
    container.innerHTML = stateEmpty('Ninguna skill coincide', `No hay skills activas que coincidan con «${ctx.filter.trim()}».`, 'search');
    return;
  }
  container.innerHTML = `<ul class="kn-skill-grid" aria-label="Skills activas">${list.map(s => `<li class="kn-skill">
    <h3 class="kn-skill-name mono">${escapeHtml(s.name)}</h3>
    <p class="kn-skill-desc">${escapeHtml(s.description || 'Sin descripción')}</p>
    ${s.trigger ? `<p class="kn-skill-trigger"><span class="kn-dim">Se activa con</span> <span class="mono">${escapeHtml(s.trigger)}</span></p>` : ''}
    ${s.path ? `<p class="kn-skill-path mono">${escapeHtml(s.path)}</p>` : ''}
  </li>`).join('')}</ul>`;
}

export async function loadSkills() {
  const container = $('#skills-container');
  try {
    ctx.active = (await api('/api/skills')) || [];
    renderActive();
  } catch (err) {
    console.error(err);
    $('#skills-count').textContent = 'skills/';
    container.innerHTML = stateError(err.message, 'No se pudo cargar el catálogo de skills');
  }
}

// ---- Buzón ---------------------------------------------------------------------------------

function originLabel(p) {
  return ORIGIN_LABEL[p.origin] || p.origin || 'Origen desconocido';
}

function currentProposal() {
  return ctx.proposals.find(p => p.name === ctx.selected) || null;
}

// integrity explica el estado de verificación en texto, no solo con color.
function integrity(p) {
  const hashes = p.sha256 && typeof p.sha256 === 'object' ? Object.entries(p.sha256) : [];
  const list = hashes.length
    ? `<ul class="kn-hashes">${hashes.map(([file, h]) => `<li><code title="${escapeHtml(h)}">${escapeHtml(shortHash(h))}</code><span class="kn-dim mono">${escapeHtml(file)}</span></li>`).join('')}</ul>`
    : '';
  const explanation = p.verified
    ? 'La huella SHA-256 de la skill coincide con la del registro oficial: el contenido no se ha alterado.'
    : 'No se pudo contrastar con el registro oficial (por ejemplo, porque se generó en local). Revisa el contenido antes de aprobarla.';
  return `<div class="kn-integrity">${p.verified ? statusPill('ok', 'SHA-256 verificado') : statusPill('warn', 'Sin verificar')}<p class="kn-hint">${escapeHtml(explanation)}</p>${list}</div>`;
}

function renderPreview(p) {
  const created = formatDateTime(p.created_at);
  const rows = [
    ['Por qué', escapeHtml(p.justification || 'Directriz recomendada')],
    ['Detectada por', p.detected_by ? `<span class="mono">${escapeHtml(p.detected_by)}</span>` : ''],
    ['Fuente', p.source ? `<span class="mono">${escapeHtml(p.source)}</span>` : ''],
    ['Rol', p.role ? `<span class="mono">${escapeHtml(p.role)}</span>` : ''],
    ['Creada', created ? escapeHtml(created) : ''],
    ['Integridad', integrity(p)]
  ].filter(([, v]) => v);
  const md = (p.skill_md || '').trim();
  let body;
  if (!md) body = stateEmpty('Sin contenido', 'La propuesta no incluye SKILL.md.', 'file');
  else if (ctx.raw) body = `<pre class="md-raw kn-md-box" tabindex="0" aria-label="SKILL.md en Markdown">${escapeHtml(p.skill_md)}</pre>`;
  else body = `<div class="md kn-md-box" tabindex="0" aria-label="SKILL.md renderizado">${renderMarkdown(p.skill_md)}</div>`;

  return `<article class="kn-preview" aria-labelledby="pv-h">
    <div class="kn-preview-head">
      <h3 id="pv-h" class="mono">${escapeHtml(p.name)}</h3>
      ${badge(p.origin === 'midudev' ? 'accent' : 'neutral', originLabel(p))}
      <span class="grow"></span>
      <button type="button" class="btn btn-danger" data-action="reject" data-name="${escapeHtml(p.name)}">${icon('x', 16)}Rechazar</button>
      <button type="button" class="btn btn-primary" data-action="approve" data-name="${escapeHtml(p.name)}">${icon('check', 16)}Aprobar</button>
    </div>
    <dl class="kn-dl">${rows.map(([k, v]) => `<dt>${escapeHtml(k)}</dt><dd>${v}</dd>`).join('')}</dl>
    <div class="kn-doc">
      <div class="kn-doc-head">
        <span class="mono">SKILL.md</span>
        <div class="seg-group seg-inline" role="group" aria-label="Formato de SKILL.md">
          <button type="button" class="seg seg-wide" data-format="view" aria-pressed="${!ctx.raw}">Vista</button>
          <button type="button" class="seg seg-wide" data-format="raw" aria-pressed="${ctx.raw}">Markdown</button>
        </div>
      </div>
      ${body}
    </div>
  </article>`;
}

function renderInbox() {
  const container = $('#inbox-container');
  const count = $('#inbox-count');
  count.textContent = plural(ctx.proposals.length, 'pendiente', 'pendientes');
  if (ctx.proposals.length === 0) {
    container.innerHTML = `<div class="kn-pad">${stateEmpty('Buzón vacío', 'No hay propuestas pendientes. Escanea el stack para descubrir nuevas skills.', 'inbox')}</div>`;
    return;
  }
  if (!currentProposal()) ctx.selected = ctx.proposals[0].name;
  const items = ctx.proposals.map((p) => {
    const current = p.name === ctx.selected;
    const created = formatDateTime(p.created_at);
    return `<li><button type="button" class="kn-pp" data-pick="${escapeHtml(p.name)}" aria-pressed="${current}">
      <span class="kn-pp-top"><span class="mono kn-pp-name">${escapeHtml(p.name)}</span>${p.role ? `<span class="mono kn-dim">${escapeHtml(p.role)}</span>` : ''}</span>
      <span class="kn-pp-sub">${escapeHtml(originLabel(p))}${p.verified ? ' · SHA-256 verificado' : ' · sin verificar'}</span>
      ${created ? `<span class="kn-dim kn-pp-date">${escapeHtml(created)}</span>` : ''}
    </button></li>`;
  }).join('');
  container.innerHTML = `<div class="kn-split">
    <ul class="kn-plist" aria-label="Propuestas pendientes">${items}</ul>
    ${renderPreview(currentProposal())}
  </div>`;
}

export async function loadSkillsInbox() {
  const container = $('#inbox-container');
  try {
    ctx.proposals = (await api('/api/skills/inbox')) || [];
    renderInbox();
  } catch (err) {
    console.error(err);
    $('#inbox-count').textContent = 'No disponible';
    container.innerHTML = `<div class="kn-pad">${stateError(err.message, 'No se pudo consultar el buzón')}</div>`;
  }
}

// reloadAll refresca el buzón, la lista activa y la insignia de la barra lateral.
async function reloadAll() {
  await Promise.all([loadSkillsInbox(), loadSkills(), refreshOverview(['inbox'])]);
}

// ---- Acciones ------------------------------------------------------------------------------------

async function scanSkills() {
  const btn = $('#btn-scan-skills');
  const restore = setBusy(btn, 'Escaneando…');
  feedback('info', 'Analizando el stack tecnológico, las dependencias y los patrones del repositorio. Puede tardar unos segundos.');
  try {
    const report = await api('/api/skills/scan', { body: { offline: false } });
    const techs = Array.isArray(report.detected_technologies) && report.detected_technologies.length
      ? report.detected_technologies.join(', ')
      : 'ninguna';
    const added = Array.isArray(report.skills_proposed) ? report.skills_proposed.length : 0;
    feedback('ok', `<strong>Escaneo completado.</strong> Tecnologías detectadas: ${escapeHtml(techs)}. Propuestas nuevas: ${formatNumber(added)} (minadas en local: ${formatNumber(report.mined_skills_count || 0)}, del registro: ${formatNumber(report.registry_skills_count || 0)}). Pendientes en el buzón: ${formatNumber(report.total_in_inbox || 0)}.`);
    await reloadAll();
  } catch (err) {
    console.error(err);
    feedback('bad', `<strong>El escaneo ha fallado.</strong> ${escapeHtml(err.message)}`);
  } finally {
    restore();
  }
}

// afterDecision devuelve el foco al buzón: la propuesta que queda seleccionada o,
// con el buzón vacío, el botón de escanear.
function afterDecision() {
  const target = $('#inbox-container [aria-pressed="true"][data-pick]') || $('#btn-scan-skills');
  if (target) target.focus();
}

async function approve(name) {
  const ok = await confirmDialog({
    title: 'Aprobar skill',
    message: `¿Aprobar e instalar la skill «${name}» en skills/? Pasará a estar disponible para los agentes.`,
    confirmLabel: 'Aprobar e instalar'
  });
  if (!ok) return;
  try {
    const res = await api('/api/skills/approve', { body: { name } });
    const message = (res && res.message) || `Skill «${name}» aprobada e instalada.`;
    // El servidor añade «(aviso: …)» cuando la skill queda instalada pero falla algo
    // secundario (p. ej. regenerar el índice): no es un error, pero se muestra.
    const warned = /\(aviso:/i.test(message);
    feedback(warned ? 'warn' : 'ok', escapeHtml(message));
    toast(message, warned ? 'warn' : 'ok');
    await reloadAll();
    afterDecision();
  } catch (err) {
    feedback('bad', `<strong>No se pudo aprobar la skill.</strong> ${escapeHtml(err.message)}`);
    toast(`Error aprobando la skill: ${err.message}`, 'bad');
  }
}

async function reject(name) {
  const ok = await confirmDialog({
    title: 'Rechazar propuesta',
    message: `¿Rechazar la propuesta «${name}»? Se descartará del buzón y no se instalará.`,
    confirmLabel: 'Rechazar',
    danger: true
  });
  if (!ok) return;
  try {
    const res = await api('/api/skills/reject', { body: { name } });
    feedback('ok', escapeHtml((res && res.message) || `Propuesta «${name}» descartada.`));
    await reloadAll();
    afterDecision();
  } catch (err) {
    feedback('bad', `<strong>No se pudo rechazar la propuesta.</strong> ${escapeHtml(err.message)}`);
    toast(`Error rechazando la propuesta: ${err.message}`, 'bad');
  }
}

// ---- Montaje -------------------------------------------------------------------------------------

export function initSkills() {
  $('#btn-scan-skills').addEventListener('click', scanSkills);

  $('#inbox-container').addEventListener('click', (event) => {
    const pick = event.target.closest('[data-pick]');
    if (pick) {
      ctx.selected = pick.getAttribute('data-pick');
      ctx.raw = false;
      renderInbox();
      const again = $(`#inbox-container [data-pick="${CSS.escape(ctx.selected)}"]`);
      if (again) again.focus();
      return;
    }
    const format = event.target.closest('[data-format]');
    if (format) {
      ctx.raw = format.getAttribute('data-format') === 'raw';
      renderInbox();
      const again = $(`#inbox-container [data-format="${format.getAttribute('data-format')}"]`);
      if (again) again.focus();
      return;
    }
    const action = event.target.closest('[data-action]');
    if (!action) return;
    const name = action.getAttribute('data-name');
    if (action.getAttribute('data-action') === 'approve') approve(name);
    else reject(name);
  });

  $('#skills-filter').addEventListener('input', (event) => {
    ctx.filter = event.target.value;
    renderActive();
  });
}
