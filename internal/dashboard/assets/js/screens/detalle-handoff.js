// Pestaña «Handoff» del detalle (D4, esfuerzo mínimo): muestra el único handoff.md
// del incremento con sus metadatos y las cinco secciones en Markdown, o un estado
// vacío, y conserva el modal «Crear handoff». Se retirará en una versión próxima.

import { api, escapeHtml } from '../core/api.js';
import { $, button, closeDialog, openDialog, setBusy, stateEmpty, stateError, stateLoading, statusPill, toast } from '../core/components.js';
import { icon } from '../core/icons.js';
import { renderMarkdown } from '../core/markdown.js';
import { state } from '../core/state.js';

const SECTIONS = [
  ['executive_summary', 'Resumen ejecutivo'],
  ['artifacts', 'Artefactos modificados y creados'],
  ['decisions', 'Decisiones técnicas y acuerdos'],
  ['risks_and_blockers', 'Riesgos, bloqueos y preguntas abiertas'],
  ['direct_instructions', 'Instrucciones directas para el siguiente rol']
];

function statusBadge(status) {
  switch (status) {
    case 'ready': return statusPill('ok', 'Listo');
    case 'in-progress': return statusPill('accent', 'En progreso');
    case 'blocked': return statusPill('bad', 'Bloqueado');
    default: return statusPill('neutral', status || 'Sin estado');
  }
}

function formatTimestamp(ts) {
  if (!ts) return '';
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) return '';
  try {
    return d.toLocaleString('es-ES');
  } catch (_) {
    return ts;
  }
}

function handoffHtml(ho) {
  const meta = ho.metadata || {};
  const sections = ho.sections || {};
  const when = formatTimestamp(meta.timestamp);
  const body = SECTIONS.map(([key, title]) => {
    const text = (sections[key] || '').trim();
    return `<section class="dt-ho-section"><h3 class="dt-label">${escapeHtml(title)}</h3>${text ? `<div class="md">${renderMarkdown(text)}</div>` : '<p class="muted">Sin contenido.</p>'}</section>`;
  }).join('');
  return `<article class="card dt-ho" aria-labelledby="dt-ho-title">
    <header class="dt-ho-head">
      <h2 id="dt-ho-title" class="sr-only">Handoff de ${escapeHtml(meta.from_phase || 'origen')} a ${escapeHtml(meta.to_phase || 'destino')}</h2>
      <span class="dt-flow"><span class="dt-phase">${escapeHtml(meta.from_phase || 'origen')}</span>${icon('arrowRight', 14)}<span class="dt-phase is-target">${escapeHtml(meta.to_phase || 'destino')}</span></span>
      <span class="muted"><span class="mono">${escapeHtml(meta.from_role || '—')}</span> → <span class="mono">${escapeHtml(meta.to_role || '—')}</span></span>
      ${statusBadge(meta.status)}
      ${when ? `<time class="dim" datetime="${escapeHtml(meta.timestamp)}">${escapeHtml(when)}</time>` : ''}
    </header>
    <div class="dt-ho-body">${body}</div>
  </article>`;
}

function createButton() {
  return `<div class="dt-ho-actions">${button({ label: 'Crear handoff', iconName: 'plus', attrs: 'data-action="create-handoff" title="Redactar un nuevo relevo formal"' })}</div>`;
}

// renderHandoff carga y pinta el handoff del cambio. El archivado no admite crear
// otro (el servidor solo escribe en cambios activos).
export async function renderHandoff(panel, change, { archived, isCurrent }) {
  panel.innerHTML = stateLoading('Cargando el handoff…');
  const actions = archived ? '' : createButton();
  try {
    const ho = await api(`/api/handoffs?change=${encodeURIComponent(change)}`);
    if (!isCurrent()) return;
    panel.innerHTML = actions + handoffHtml(ho);
  } catch (err) {
    if (!isCurrent()) return;
    // El servidor responde 404 tanto si falta handoff.md como si no se puede leer: solo lo primero es «vacío».
    if (err.status === 404 && /no existe handoff.md/i.test(err.message)) {
      panel.innerHTML = actions + stateEmpty('Este incremento aún no tiene un handoff', 'Un handoff deja por escrito el relevo entre fases o roles (handoff.md).', 'handoff');
    } else {
      panel.innerHTML = actions + stateError(err.message, 'No se pudo cargar el handoff');
    }
  }
}

// ---- Modal «Crear handoff» ---------------------------------------------------------------

function fillChangeOptions(selected) {
  const select = $('#handoff-modal-change');
  select.innerHTML = state.increments
    .filter(inc => inc.type === 'active')
    .map(inc => `<option value="${escapeHtml(inc.name)}">${escapeHtml(inc.name)} (fase: ${escapeHtml(inc.phase)})</option>`)
    .join('');
  if (selected) select.value = selected;
}

// initHandoff enlaza el botón (delegado, porque el panel se regenera) y el envío.
export function initHandoff({ panel, getChange, onCreated }) {
  const dialog = $('#modal-create-handoff');
  const errorBox = $('#handoff-error');
  const submit = $('#btn-submit-create-handoff');
  const val = (id) => $(id).value.trim();

  panel.addEventListener('click', (event) => {
    if (!event.target.closest('[data-action="create-handoff"]')) return;
    fillChangeOptions(getChange());
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
      onCreated(change);
    } catch (err) {
      errorBox.hidden = false;
      errorBox.textContent = `Error al registrar el handoff: ${err.message}`;
    } finally {
      restore();
    }
  });
}
