// Pestaña «Pruebas» del detalle: muestra la batería de pruebas de aceptación (pruebas.md) con su
// cobertura y sus hallazgos y, cuando el servidor lo permite (acceptance_tests.marks.available), la
// matriz de marcas casos x entornos con el formulario para marcar y el bloque «Publicar marcas».
// Sin edición de pruebas.md ni ejecución de casos: solo se añaden marcas a pruebas-estado.yaml y se
// publican tras una confirmación explícita. Todo texto proveniente de pruebas.md, de las marcas o de
// un rechazo se inserta escapado, nunca como HTML ni como Markdown.

import { api, escapeHtml } from '../core/api.js';
import { badge, confirmDialog, stateEmpty, stateError, toast } from '../core/components.js';

const COLUMNS = [
  ['id', 'ID'], ['role', 'Rol'], ['type', 'Tipo'], ['priority', 'Prioridad'],
  ['preconditions', 'Precondiciones'], ['steps', 'Pasos'], ['expected', 'Resultado esperado'],
  ['cubre', 'Cubre'], ['environments', 'Entornos'],
];

const RESULTS = ['ok', 'ko', 'blocked', 'n/a'];
const STATE_KIND = { ok: 'ok', ko: 'bad', blocked: 'warn', 'n/a': 'neutral', stale: 'warn', unmarked: 'neutral' };
const STATE_LABEL = { ok: 'ok', ko: 'ko', blocked: 'bloqueada', 'n/a': 'n/a', stale: 'caducada', unmarked: 'sin marcar' };

function coverage(c) {
  const cov = c || {};
  return `${cov.covered || 0}/${cov.total || 0}`;
}

function summaryHtml(at) {
  return `<p class="ix-more" role="status">Escenarios ${escapeHtml(coverage(at.scenarios))} · `
    + `Requisitos ${escapeHtml(coverage(at.requirements))} · Casos ${escapeHtml(String(at.case_count || 0))}</p>`;
}

function findingsHtml(findings) {
  if (!findings.length) return '<p class="ix-more">Sin hallazgos.</p>';
  const items = findings.map(f => {
    const where = f.line ? `${f.file}:${f.line}` : f.file;
    return `<li><code>${escapeHtml(f.code)}</code> ${escapeHtml(where)}: ${escapeHtml(f.message)}</li>`;
  }).join('');
  return `<ul class="at-findings">${items}</ul>`;
}

function casesHtml(cases) {
  if (!cases.length) return stateEmpty('La tabla no tiene casos');
  const head = COLUMNS.map(([, label]) => `<th scope="col">${escapeHtml(label)}</th>`).join('');
  const rows = cases.map(c => `<tr>${COLUMNS.map(([key]) => `<td>${escapeHtml(c[key] || '')}</td>`).join('')}</tr>`).join('');
  return `<div class="table-wrap"><table class="data-table"><thead><tr>${head}</tr></thead><tbody>${rows}</tbody></table></div>`;
}

// refusalText compone el rechazo tipado: etiqueta, motivo y continuación (todo texto del servidor).
function refusalText(r) {
  if (!r) return '';
  return [r.label, r.error, r.continuation].filter(Boolean).join(' · ');
}

function refusalHtml(r) {
  if (!r) return '';
  const label = r.label ? `<strong>${escapeHtml(r.label)}</strong> ` : '';
  const cont = r.continuation ? ` <span class="ix-more">${escapeHtml(r.continuation)}</span>` : '';
  return `<p class="ix-more" role="alert">${label}<code>${escapeHtml(r.code || '')}</code> ${escapeHtml(r.error || '')}${cont}</p>`;
}

function entryText(e) {
  return `${e.case} · ${e.environment} · ${e.result} · ${e.by} · ${e.at}`;
}

function cellHtml(cell, marks) {
  const state = cell.state || 'unmarked';
  let html = badge(STATE_KIND[state] || 'neutral', STATE_LABEL[state] || state);
  const cur = cell.current;
  if (cur) {
    if (cur.note) html += `<div class="ix-more">${escapeHtml(cur.note)}</div>`;
    if (cur.commit) html += `<div class="ix-more mono">${escapeHtml(cur.commit)}</div>`;
  }
  if (state === 'stale' && cell.stale) {
    html += `<div class="ix-more">caducada: ${escapeHtml(cell.stale.result)}</div>`;
  }
  const options = RESULTS.map(r => `<option value="${escapeHtml(r)}">${escapeHtml(r)}</option>`).join('');
  html += `<form class="at-mark-form" data-case="${escapeHtml(cell.case)}" data-env="${escapeHtml(cell.environment)}">`
    + `<select name="result" aria-label="Resultado">${options}</select>`
    + `<input type="text" name="commit" maxlength="${escapeHtml(String(marks.commitMax || 100))}" placeholder="commit" aria-label="Commit">`
    + `<input type="text" name="note" maxlength="${escapeHtml(String(marks.noteMax || 500))}" placeholder="nota" aria-label="Nota">`
    + '<button type="submit" class="btn btn-sm">Marcar</button></form>';
  return html;
}

function matrixHtml(cases, marks) {
  const envs = marks.environments || [];
  if (!cases.length || !envs.length) return '';
  const byKey = new Map();
  (marks.cells || []).forEach(c => byKey.set(`${c.case}\u0000${c.environment}`, c));
  const head = `<th scope="col">Caso</th>${envs.map(e => `<th scope="col">${escapeHtml(e)}</th>`).join('')}`;
  const rows = cases.map(c => {
    const tds = envs.map(env => {
      const cell = byKey.get(`${c.id}\u0000${env}`);
      // Celda no aplicable (el caso no declara ese entorno): sin control.
      return `<td>${cell ? cellHtml(cell, marks) : '—'}</td>`;
    }).join('');
    return `<tr><th scope="row">${escapeHtml(c.id)}</th>${tds}</tr>`;
  }).join('');
  return `<div class="table-wrap"><table class="data-table at-marks"><thead><tr>${head}</tr></thead><tbody>${rows}</tbody></table></div>`;
}

function publishHtml(marks) {
  const pending = marks.pending || [];
  const list = pending.length
    ? `<ul class="at-pending">${pending.map(e => `<li>${escapeHtml(entryText(e))}</li>`).join('')}</ul>` : '';
  const blocked = pending.length === 0 || !!marks.pendingUnavailable;
  return '<section class="at-publish" aria-label="Publicar marcas"><h3>Publicar marcas</h3>'
    + `<p class="ix-more" role="status">Remoto: ${escapeHtml(marks.remote || '')} · Rama: ${escapeHtml(marks.branch || '')} · `
    + `${escapeHtml(String(pending.length))} entradas pendientes</p>`
    + refusalHtml(marks.pendingUnavailable) + list
    + `<button type="button" class="btn" data-marks-publish${blocked ? ' disabled' : ''}>Publicar marcas</button></section>`;
}

function marksHtml(cases, marks) {
  if (!marks) return '';
  if (!marks.available) {
    return marks.unavailable
      ? stateEmpty('Marcas no disponibles', `${marks.unavailable.error || ''} ${marks.unavailable.continuation || ''}`.trim())
      : '';
  }
  let html = '';
  if (marks.error) html += stateError(marks.error, 'No se pudieron leer las marcas');
  if (marks.summaryText) html += `<p class="ix-more" role="status">${escapeHtml(marks.summaryText)}</p>`;
  if (marks.error) return html;
  return html + matrixHtml(cases, marks) + publishHtml(marks);
}

// bindMarks enlaza el formulario de marcas y el botón de publicar (delegación en el panel).
function bindMarks(panel, ctx) {
  const repaint = marks => {
    ctx.at.marks = marks;
    paint(panel, ctx);
  };
  const fail = err => {
    const d = (err && err.data) || {};
    toast(refusalText({ label: d.label, error: d.error || (err && err.message), continuation: d.continuation }), 'bad');
  };
  panel.onsubmit = async ev => {
    const form = ev.target.closest('form.at-mark-form');
    if (!form) return;
    ev.preventDefault();
    const body = {
      change: ctx.change, case: form.dataset.case, environment: form.dataset.env,
      result: form.elements.result.value,
    };
    if (form.elements.commit.value) body.commit = form.elements.commit.value;
    if (form.elements.note.value) body.note = form.elements.note.value;
    try {
      const res = await api('/api/increments/marks', { body });
      toast('Marca añadida');
      if (res && res.marks) repaint(res.marks);
    } catch (err) { fail(err); }
  };
  panel.onclick = async ev => {
    const btn = ev.target.closest('[data-marks-publish]');
    if (!btn || btn.disabled) return;
    const m = ctx.at.marks;
    const pending = m.pending || [];
    const message = `Remoto: ${m.remote} · Rama: ${m.branch} · ${pending.length} entradas pendientes\n`
      + pending.map(entryText).join('\n');
    const ok = await confirmDialog({ title: 'Publicar marcas', message, confirmLabel: 'Publicar' });
    if (!ok) return;
    try {
      const res = await api('/api/increments/marks/publish', { body: { change: ctx.change, confirm: true, branch: m.branch } });
      const where = res.commit ? `${res.commit} en ${res.remote}/${res.branch}` : `${res.remote}/${res.branch}`;
      toast(`Marcas: ${res.status} (${where})`);
      (res.uncommittedDocs || []).forEach(d => toast(`Documento sin publicar: ${d}`, 'warn'));
      if (res.marks) repaint(res.marks);
    } catch (err) { fail(err); }
  };
}

function paint(panel, ctx) {
  const at = ctx.at;
  let html = '';
  if (at.error) html += stateError(at.error, 'No se pudo calcular la batería de pruebas');
  if (at.warning) html += `<p class="ix-more" role="status">${escapeHtml(at.warning)}</p>`;
  if (!at.present) {
    html += stateEmpty(at.active ? 'Todavía no hay batería de pruebas' : 'No hay batería de pruebas');
    if (at.findings && at.findings.length) html += findingsHtml(at.findings);
    panel.innerHTML = html;
    return;
  }
  if (!at.error) {
    const cases = at.cases || [];
    html += summaryHtml(at) + marksHtml(cases, at.marks) + findingsHtml(at.findings || []) + casesHtml(cases);
  }
  panel.innerHTML = html;
}

// renderAcceptanceTests pinta la pestaña a partir de detail.acceptance_tests (nil: no activo).
export function renderAcceptanceTests(panel, { detail, change }) {
  const at = detail.acceptance_tests;
  if (!at || (!at.active && !at.present && !at.error)) {
    panel.innerHTML = stateEmpty('Las pruebas de aceptación no están activadas para este proyecto.');
    return;
  }
  const ctx = { at, change };
  paint(panel, ctx);
  bindMarks(panel, ctx);
}
